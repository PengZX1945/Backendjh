// Package apperr 定义业务异常。
//
// 约定：service 层遇到可预期的失败（参数不合法、越权、状态不允许……）时，
// 返回 *apperr.Error 而不是裸 error；handler 只需把它交给 gin 的 error 收集器，
// 由统一响应中间件翻译成 {code, msg, data}。底层不可预期的 error 原样上抛，
// 中间件会兜底成 114，避免把数据库细节泄露给客户端。
package apperr

import (
	"errors"

	"lostfound/pkg/errcode"
)

// Error 是带业务错误码的异常。
type Error struct {
	// Code 决定对外返回的 code/msg/HTTP 状态码。
	Code errcode.Code
	// Cause 保留底层原因，仅用于写日志，不会返回给客户端。
	Cause error
}

// New 构造一个不含底层原因的业务异常。
func New(code errcode.Code) *Error {
	return &Error{Code: code}
}

// Wrap 构造一个附带底层原因的业务异常。
// 用于「外部失败值得归入某个业务码，但真实原因仍需留痕」的场景，例如把
// JWT 解析失败、JSON 解析失败归为参数错误的同时保留原始报错。
func Wrap(code errcode.Code, cause error) *Error {
	return &Error{Code: code, Cause: cause}
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Code.Msg()
}

// Unwrap 支持 errors.Is / errors.As 沿链查找。
func (e *Error) Unwrap() error {
	return e.Cause
}

// From 从任意 error 中提取业务异常。
// 命中 *Error（含被 Wrap 的层级）则原样返回；否则归为服务器内部错误并保留原因。
func From(err error) *Error {
	var business *Error
	if errors.As(err, &business) {
		return business
	}
	return Wrap(errcode.ServerError, err)
}
