package main

import (
	"github.com/kshdb/plugin_go"
)

// 创建要导出的对象
type MyPlugin struct{}

// 导出的方法，带有rpc签名
func (p *MyPlugin) SayHello(name string, msg *string) error {
	*msg = "在插件里执行==>>:Hello, " + name
	return nil
}

// 导出的方法，带有rpc签名
func (p *MyPlugin) SayHello1(name map[string]any, msg *map[string]any) error {
	*msg = name
	return nil
}

func main() {
	plugin := &MyPlugin{}
	// 注册要导出的对象
	plugin_go.Register(plugin)
	// 运行主程序
	plugin_go.Run()
}
