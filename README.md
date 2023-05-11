# plugin_go: Plugins for Go

golang纯标准库库实现基于rpc的插件化

## Example

1.插件创建

```go

package main

import "github.com/kshdb/plugin_go"

// 创建一个插件对象
type MyPlugin struct{}

// 实现插件方法SayHello
func (p *MyPlugin) SayHello(name string, msg *string) error {
    *msg = "Hello, " + name
    return nil
}

func main() {
	plugin := &MyPlugin{}
	// 注册插件对象
	plugin_go.Register(plugin)
	// 运行插件对象
	plugin_go.Run()
}
```

2.插件调用

```go
package main

import (
	"log"
	"github.com/kshdb/plugin_go"
)

func main() {
	//调用插件
	p := plugin_go.NewPlugin("tcp", "plugins/hello-world/hello-world")
	p.Start()
	defer p.Stop()
	var resp string
    //调用插件函数
	if err := p.Call("MyPlugin.SayHello", "Go developer", &resp); err != nil {
		log.Print(err)
	} else {
		log.Print(resp)
	}
}
```


## Bugs

Report bugs in Github.  Pull requests are welcome!

## TODO

* Automatically restart crashed plugins
* Automatically switch between ```unix``` and ```TCP``` if setup of one fails

## License

MIT
