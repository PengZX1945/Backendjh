package errcode

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Error) Error() string { return e.Msg }

func New(code int, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

var (
	Success          = New(0, "success")
	ParamError       = New(10001, "参数错误")
	Unauthorized     = New(10002, "未登录或登录已过期")
	Forbidden        = New(10003, "无权限操作")
	NotFound         = New(10004, "资源不存在")
	UsernameExists   = New(10005, "用户名已存在")
	WrongPassword    = New(10006, "用户名或密码错误")
	StatusNotAllowed = New(10007, "当前状态不允许该操作")
	UploadFailed     = New(10008, "文件上传失败")
	DuplicateSubmit  = New(10009, "请勿重复提交")
	AccountDisabled  = New(10010, "账号已被禁用")
	InternalError    = New(20001, "服务器内部错误")
	SamePassword     = New(10011, "新密码不能与旧密码相同")
)
