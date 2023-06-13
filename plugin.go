package plugin_go

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	errInvalidMessage      = ErrInvalidMessage(errors.New("Invalid ready message"))
	errRegistrationTimeout = ErrRegistrationTimeout(errors.New("Registration timed out"))
)

// 插件结构对象
type Plugin struct {
	exe         string
	proto       string
	unixdir     string
	params      []string
	initTimeout time.Duration
	exitTimeout time.Duration
	handler     ErrorHandler
	running     bool
	meta        meta
	objsCh      chan *objects
	connCh      chan *conn
	killCh      chan *waiter
	exitCh      chan struct{}
}

// 新建插件对象
func NewPlugin(proto, path string, params ...string) *Plugin {
	if proto != "unix" && proto != "tcp" {
		panic("Invalid protocol. Specify 'unix' or 'tcp'.")
	}
	p := &Plugin{
		exe:         path,
		proto:       proto,
		params:      params,
		initTimeout: 2 * time.Second,
		exitTimeout: 2 * time.Second,
		handler:     NewDefaultErrorHandler(),
		meta:        meta("plugin_go" + randstr(11)),
		objsCh:      make(chan *objects),
		connCh:      make(chan *conn),
		killCh:      make(chan *waiter),
		exitCh:      make(chan struct{}),
	}
	return p
}

// 设置错误（和输出）处理程序实现
func (p *Plugin) SetErrorHandler(h ErrorHandler) {
	if p.running {
		panic("Cannot call SetErrorHandler after Start")
	}
	p.handler = h
}

func (p *Plugin) SetTimeout(t time.Duration) {
	if p.running {
		panic("Cannot call SetTimeout after Start")
	}
	if t == 0 {
		return
	}
	p.initTimeout = t
	p.exitTimeout = t
}

func (p *Plugin) SetSocketDirectory(dir string) {
	if p.running {
		panic("Cannot call SetSocketDirectory after Start")
	}
	p.unixdir = dir
}

func (p *Plugin) String() string {
	return fmt.Sprintf("%s %s", p.exe, strings.Join(p.params, " "))
}

func (p *Plugin) Start() {
	p.running = true
	go p.run()
}

func (p *Plugin) Stop() {
	wr := newWaiter()
	p.killCh <- wr
	wr.wait()
	p.exitCh <- struct{}{}
}

func (p *Plugin) Call(name string, args interface{}, resp interface{}) error {
	conn := &conn{wr: newWaiter()}
	p.connCh <- conn
	conn.wr.wait()

	if conn.err != nil {
		return conn.err
	}

	return conn.client.Call(name, args, resp)
}

func (p *Plugin) Objects() ([]string, error) {
	objects := &objects{wr: newWaiter()}
	p.objsCh <- objects
	objects.wr.wait()

	return objects.list, objects.err
}

type ErrorHandler interface {
	// Error is called whenever a non-fatal error occurs in the plugin subprocess.
	Error(error)
	// Print is called for each line of output received from the plugin subprocess.
	Print(interface{})
}

type DefaultErrorHandler struct{}

func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{}
}

func (e *DefaultErrorHandler) Error(err error) {
	log.Print("error: ", err)
}

func (e *DefaultErrorHandler) Print(s interface{}) {
	log.Print(s)
}

const internalObject = "PingoRpc"

type conn struct {
	client *rpc.Client
	err    error
	wr     *waiter
}

type waiter struct {
	c chan struct{}
}

func newWaiter() *waiter {
	return &waiter{c: make(chan struct{})}
}

func (wr *waiter) wait() {
	<-wr.c
}

func (wr *waiter) done() {
	close(wr.c)
}

func (wr *waiter) reset() {
	wr.c = make(chan struct{})
}

type client struct {
	*rpc.Client
	secret string
}

func newClient(s string, conn io.ReadWriteCloser) *client {
	return &client{secret: s, Client: rpc.NewClient(conn)}
}

func (c *client) authenticate(w io.Writer) error {
	_, err := io.WriteString(w, "Auth-Token: "+c.secret+"\n\n")
	return err
}

func dialAuthRpc(secret, network, address string, timeout time.Duration) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	c := newClient(secret, conn)
	if err := c.authenticate(conn); err != nil {
		return nil, err
	}
	return c.Client, nil
}

type objects struct {
	list []string
	err  error
	wr   *waiter
}

type ctrl struct {
	p    *Plugin
	objs []string
	// Protocol and address for RPC
	proto, addr string
	// Secret needed to connect to server
	secret string
	// Unrecoverable error is used as response to calls after it happened.
	err error
	// This channel is an alias to p.connCh. It allows to
	// intermittedly process calls (only when we can handle them).
	connCh chan *conn
	// Same as above, but for objects requests
	objsCh chan *objects
	// Timeout on plugin startup time
	timeoutCh <-chan time.Time
	// Get notification from Wait on the subprocess
	waitCh chan error
	// Get output lines from subprocess
	linesCh chan string
	// Respond to a routine waiting for this mail loop to exit.
	over *waiter
	// Executable
	proc *os.Process
	// RPC client to subprocess
	client *rpc.Client
}

func newCtrl(p *Plugin, t time.Duration) *ctrl {
	return &ctrl{
		p:         p,
		timeoutCh: time.After(t),
		linesCh:   make(chan string),
		waitCh:    make(chan error),
	}
}

func (c *ctrl) fatal(err error) {
	c.err = err
	c.open()
	c.kill()
}

func (c *ctrl) isFatal() bool {
	return c.err != nil
}

func (c *ctrl) close() {
	c.connCh = nil
	c.objsCh = nil
}

func (c *ctrl) open() {
	c.connCh = c.p.connCh
	c.objsCh = c.p.objsCh
}

func (c *ctrl) ready(val string) bool {
	var err error

	if err := c.parseReady(val); err != nil {
		c.fatal(err)
		return false
	}

	c.client, err = dialAuthRpc(c.secret, c.proto, c.addr, c.p.initTimeout)
	if err != nil {
		c.fatal(err)
		return false
	}

	// Remove the temp socket now that we are connected
	if c.proto == "unix" {
		if err := os.Remove(c.addr); err != nil {
			c.p.handler.Error(errors.New("Cannot remove temporary socket: " + err.Error()))
		}
	}

	// Defuse the timeout on ready
	c.timeoutCh = nil

	return true
}

func (c *ctrl) readOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		c.linesCh <- scanner.Text()
	}
}

func (c *ctrl) waitErr(pidCh chan<- int, err error) {
	close(pidCh)
	c.waitCh <- err
}

func (c *ctrl) wait(pidCh chan<- int, exe string, params ...string) {
	defer close(c.waitCh)

	cmd := exec.Command(exe, params...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.waitErr(pidCh, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.waitErr(pidCh, err)
		return
	}
	if err := cmd.Start(); err != nil {
		c.waitErr(pidCh, err)
		return
	}

	pidCh <- cmd.Process.Pid
	close(pidCh)

	c.readOutput(stdout)
	c.readOutput(stderr)

	c.waitCh <- cmd.Wait()
}

func (c *ctrl) kill() {
	if c.proc == nil {
		return
	}
	// Ignore errors here because Kill might have been called after
	// process has ended.
	c.proc.Kill()
	c.proc = nil
}

func (c *ctrl) parseReady(str string) error {
	if !strings.HasPrefix(str, "proto=") {
		return errInvalidMessage
	}
	str = str[6:]
	s := strings.IndexByte(str, ' ')
	if s < 0 {
		return errInvalidMessage
	}
	proto := str[0:s]
	if proto != "unix" && proto != "tcp" {
		return errInvalidMessage
	}
	c.proto = proto

	str = str[s+1:]
	if !strings.HasPrefix(str, "addr=") {
		return errInvalidMessage
	}
	c.addr = str[5:]

	return nil
}

func (c *ctrl) objects() []string {
	list := make([]string, len(c.objs)-1)
	for i, j := 0, 0; i < len(c.objs); i++ {
		if c.objs[i] == internalObject {
			continue
		}
		list[j] = c.objs[i]
		j = j + 1
	}
	return list
}

func (p *Plugin) run() {
	if p.unixdir == "" {
		p.unixdir = os.TempDir()
	}

	params := []string{
		"-plugin_go:prefix=" + string(p.meta),
		"-plugin_go:proto=" + p.proto,
	}
	if p.proto == "unix" && p.unixdir != "" {
		params = append(params, "-plugin_go:unixdir="+p.unixdir)
	}
	for i := 0; i < len(p.params); i++ {
		params = append(params, p.params[i])
	}

	c := newCtrl(p, p.initTimeout)

	pidCh := make(chan int)
	go c.wait(pidCh, p.exe, params...)
	pid := <-pidCh

	if pid != 0 {
		if proc, err := os.FindProcess(pid); err == nil {
			c.proc = proc
		}
	}

	for {
		select {
		case <-c.timeoutCh:
			c.fatal(errRegistrationTimeout)
		case r := <-c.connCh:
			if c.isFatal() {
				r.err = c.err
				r.wr.done()
				continue
			}

			r.client = c.client
			r.wr.done()
		case o := <-c.objsCh:
			if c.isFatal() {
				o.err = c.err
				o.wr.done()
				continue
			}

			o.list = c.objects()
			o.wr.done()
		case line := <-c.linesCh:
			key, val := p.meta.parse(line)
			switch key {
			case "auth-token":
				c.secret = val
			case "fatal":
				if err := parseError(val); err != nil {
					c.fatal(err)
				} else {
					c.fatal(errors.New(val))
				}
			case "error":
				if err := parseError(val); err != nil {
					p.handler.Print(err)
				} else {
					p.handler.Print(errors.New(val))
				}
			case "objects":
				c.objs = strings.Split(val, ", ")
			case "ready":
				if !c.ready(val) {
					continue
				}
				// Start accepting calls
				c.open()
			default:
				p.handler.Print(line)
			}
		case wr := <-p.killCh:
			if c.waitCh == nil {
				wr.done()
				continue
			}

			// If we don't accept calls, kill immediately
			if c.connCh == nil || c.client == nil {
				c.kill()
			} else {
				// Be sure to kill the process if it doesn't obey Exit.
				go func(pid int, t time.Duration) {
					<-time.After(t)

					if proc, err := os.FindProcess(pid); err == nil {
						proc.Kill()
					}
				}(pid, p.exitTimeout)

				c.client.Call(internalObject+".Exit", 0, nil)
			}

			if c.client != nil {
				c.client.Close()
			}

			// Do not accept calls
			c.close()

			// When wait on the subprocess is exited, signal back via "over"
			c.over = wr
		case err := <-c.waitCh:
			if err != nil {
				if _, ok := err.(*exec.ExitError); !ok {
					p.handler.Error(err)
				}
				c.fatal(err)
			}

			// Signal to whoever killed us (via killCh) that we are done
			if c.over != nil {
				c.over.done()
			}

			c.proc = nil
			c.waitCh = nil
			c.linesCh = nil
		case <-p.exitCh:
			return
		}
	}
}
