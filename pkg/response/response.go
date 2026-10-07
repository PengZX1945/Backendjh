// Package response 负责写出统一的响应信封 {code, msg, data}。
//
// 全站只有这一个出口：成功走 OK，失败走 Fail。handler 不允许自己拼 JSON，
// 这样信封结构、字段命名与 data 的 null 语义才有唯一保证。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lostfound/pkg/errcode"
)

// Body 是统一响应信封。
//
// 三个字段都不加 omitempty：data 必须显式为 null（而不是被省略），
// 这与接口文档的响应示例一致，前端也依赖 `data` 键的存在与否区分信封与裸对象。
type Body struct {
	Code errcode.Code `json:"code"`
	Msg  string       `json:"msg"`
	Data any          `json:"data"`
}

// OK 写出成功响应（HTTP 200，code 0）。data 传 nil 时序列化为 null。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code: errcode.Success,
		Msg:  errcode.Success.Msg(),
		Data: data,
	})
}

// Fail 按错误码写出失败响应，HTTP 状态码取自错误码表。
func Fail(c *gin.Context, code errcode.Code) {
	c.JSON(code.HTTPStatus(), Body{
		Code: code,
		Msg:  code.Msg(),
		Data: nil,
	})
}
