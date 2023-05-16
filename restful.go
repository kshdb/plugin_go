package plugin_go

/*
get请求对象
*/
type GetModel struct {
	RestfulModel
	//请求头
	Header any `json:"header,omitempty"`
}

/*
post请求对象
*/
type PostModel struct {
	RestfulModel
	//请求头
	Header any `json:"header,omitempty"`
	//请求体
	Body any `json:"body,omitempty"`
	//请求表单
	Form any `json:"form,omitempty"`
	//请求post表单
	PostForm any `json:"post_form,omitempty"`
	//请求post文件
	FileList []FormFile `json:"file_list,omitempty"`
}

/*
put请求对象
*/
type PutModel struct {
	RestfulModel
	//请求头
	Header any `json:"header,omitempty"`
	//请求体
	Body any `json:"body,omitempty"`
	//请求表单
	Form any `json:"form,omitempty"`
	//请求post表单
	PostForm any `json:"post_form,omitempty"`
	//请求post文件
	FileList []FormFile `json:"file_list,omitempty"`
}

/*
delete请求对象
*/
type DeleteModel struct {
	RestfulModel
	//请求头
	Header any `json:"header,omitempty"`
	//请求体
	Body any `json:"body,omitempty"`
}

/*
基础请求
*/
type RestfulModel struct {
	//请求url
	Url string `json:"url"`
	//请求方式
	Method string `json:"method"`
}

/*
表单文件对象
*/
type FormFile struct {
	//文件名
	FileName string `json:"file_name"`
	//文件base64字符串
	FileData string `json:"file_data"`
	//文件后缀名
	FileSuffix string `json:"file_suffix"`
}

/*------插件对象存储数据模型-----*/
/*
插件对象
*/
type DataPlugin struct {
	//插件id
	ID int `json:"id"`
	//插件名称
	PluginName string `json:"plugin_name"`
	//插件logo
	PluginLogo string `json:"plugin_logo"`
	//插件路径
	PluginPath string `json:"plugin_path"`
	//插件描述
	PluginDescribe string `json:"plugin_describe"`
	//插件接口列表
	APIList []DataAPIList `json:"api_list"`
	//插件状态 -1卸载,0安装,1启用
	PluginStatus int `json:"plugin_status"`
	//插件创建日期
	CreateTime string `json:"create_time"`
}

/*
插件接口对象
*/
type DataAPIList struct {
	//接口id
	ID int `json:"id"`
	//接口名称
	ApiName string `json:"api_name"`
	//接口描述
	ApiDescribe string `json:"api_describe"`
	//函数名称
	FuncName string `json:"func_name"`
	//接口路由
	URL string `json:"url"`
	//接口请求方式
	Method string `json:"method"`
	//接口创建日期
	CreateTime string `json:"create_time"`
}
