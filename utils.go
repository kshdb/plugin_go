package plugin_go

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
)

type meta string

func (h meta) output(key, val string) {
	fmt.Printf("%s: %s: %s\n", string(h), key, val)
}

func (h meta) parse(line string) (key, val string) {
	if line == "" {
		return
	}

	if len(line) < len(string(h)) {
		return
	}

	if line[0:len(string(h))] != string(h) {
		return
	}

	line = line[len(string(h))+2:]
	end := strings.IndexByte(line, ':')
	if end < 0 {
		return
	}

	return line[0:end], line[end+2:]
}

var _letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")

func randstr(n int) string {
	b := make([]rune, n)
	l := len(_letters)

	for i := range b {
		b[i] = _letters[rand.Intn(l)]
	}

	return string(b)
}

// 解析url参数
func GetUrlParams(_url string) (_json any) {
	//解析URL
	parsedURL, _ := url.Parse(_url)

	// 获取URL参数
	params, _ := url.ParseQuery(parsedURL.RawQuery)

	// 将参数转换为JSON对象
	jsonObj := make(map[string]interface{})
	for key, values := range params {
		if len(values) > 0 {
			jsonObj[key] = values[0]
		}
	}
	// 将JSON对象序列化为字符串
	jsonBytes, _ := json.Marshal(jsonObj)
	//转换为json对象
	json.Unmarshal(jsonBytes, &_json)
	return
}
