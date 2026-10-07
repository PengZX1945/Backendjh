// Package errcode 集中定义全局业务错误码。
//
// 设计约束（作业「共同要求」）：一个 code 唯一对应一条 msg，且唯一对应一个用于
// 承载它的 HTTP 状态码。三者的映射关系全部收敛在本文件，业务代码只引用常量，
// 不再散落任何字面量 —— 想要改一次文案或状态码，只改这里一处。
package errcode

// Code 是业务错误码。0 表示成功，其余为失败。
type Code int

const (
	Success         Code = 0   // success
	BadRequest      Code = 1   // 参数错误
	Unauthorized    Code = 2   // 未登录或登录已过期
	Forbidden       Code = 3   // 无权限操作
	NotFound        Code = 4   // 资源不存在
	UsernameTaken   Code = 5   // 用户名已存在
	BadCredentials  Code = 6   // 用户名或密码错误
	InvalidState    Code = 7   // 当前状态不允许该操作
	UploadFailed    Code = 8   // 文件上传失败
	DuplicateSubmit Code = 9   // 请勿重复提交
	AccountDisabled Code = 10  // 账号已被禁用
	SamePassword    Code = 11  // 新密码不能与旧密码相同
	ServerError     Code = 114 // 服务器内部错误
)

// messageTable 是 code → msg 的唯一真相源，与接口文档「错误码表」逐条对齐。
var messageTable = map[Code]string{
	Success:         "success",
	BadRequest:      "参数错误",
	Unauthorized:    "未登录或登录已过期",
	Forbidden:       "无权限操作",
	NotFound:        "资源不存在",
	UsernameTaken:   "用户名已存在",
	BadCredentials:  "用户名或密码错误",
	InvalidState:    "当前状态不允许该操作",
	UploadFailed:    "文件上传失败",
	DuplicateSubmit: "请勿重复提交",
	AccountDisabled: "账号已被禁用",
	SamePassword:    "新密码不能与旧密码相同",
	ServerError:     "服务器内部错误",
}

// httpStatusTable 决定该错误码以什么 HTTP 状态码承载。
//
// 之所以不让所有响应都返回 200 + 业务码：文档为多数接口标注了 4xx/5xx，前端也按
// HTTP 状态码区分「会话失效」（401）与「资源不存在」（404）并据此跳登录页或渲染空态。
var httpStatusTable = map[Code]int{
	Success:         200,
	BadRequest:      400,
	Unauthorized:    401,
	Forbidden:       403,
	NotFound:        404,
	UsernameTaken:   409,
	BadCredentials:  401,
	InvalidState:    405,
	UploadFailed:    406,
	DuplicateSubmit: 409,
	AccountDisabled: 423,
	SamePassword:    400,
	ServerError:     500,
}

// Msg 返回该错误码对应的文案。未登记的错误码回落为服务器内部错误文案。
func (c Code) Msg() string {
	if message, ok := messageTable[c]; ok {
		return message
	}
	return messageTable[ServerError]
}

// HTTPStatus 返回该错误码对应的 HTTP 状态码。未登记的错误码回落为 500。
func (c Code) HTTPStatus() int {
	if status, ok := httpStatusTable[c]; ok {
		return status
	}
	return httpStatusTable[ServerError]
}
