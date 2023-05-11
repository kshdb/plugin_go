package main

import (
	"log"

	"github.com/kshdb/plugin_go"
)

func main() {
	// 从创建的可执行文件中创建一个新插件。通过 TCP 连接到它
	p := plugin_go.NewPlugin("tcp", "plugins_demo/plugins_demo")
	// 启动插件
	p.Start()
	// 使用完插件后停止它
	defer p.Stop()
	var resp1 string
	if err := p.Call("MyPlugin.SayHello", "sssss", &resp1); err != nil {
		log.Print(err)
	} else {
		log.Print("执行结果1", resp1)
	}
	var resp map[string]any
	// 从先前创建的对象调用函数
	if err := p.Call("MyPlugin.SayHello1", map[string]any{"aaa": 123, "bbb": "sddd"}, &resp); err != nil {
		log.Print(err)
	} else {
		log.Print("执行结果2", resp)
	}
}
