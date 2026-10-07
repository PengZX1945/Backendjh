// Package middleware 存放 Gin 中间件：统一响应、鉴权、跨域、访问日志。
package middleware

import (
	"github.com/gin-gonic/gin"

	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
	"lostfound/pkg/logger"
	"lostfound/pkg/response"
)

// ErrorHandler 是「业务异常统一响应中间件」。
//
// 约定：handler 与 service 都不自己拼错误响应，只把 error 交给 gin 的错误收集器
// （c.Error(err)）后返回；本中间件在整条处理链跑完之后统一收口：
//
//   - *apperr.Error（可预期的业务失败）→ 按其错误码输出 {code, msg, data:null}，
//     HTTP 状态码取自错误码表；
//   - 其他 error（未预期异常）→ 记一条带原始原因的日志，对外只回 114，
//     不把数据库或内部细节泄露给客户端。
//
// 注册位置必须在所有路由之前，这样它才能包住整条链、看到任意一处抛出的错误。
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		// 响应已经写出时不重复处理，否则会拼出两段 JSON 破坏信封结构。
		if c.Writer.Written() {
			return
		}

		failure := apperr.From(c.Errors.Last().Err)
		if failure.Code == errcode.ServerError {
			logger.Error("处理请求时发生未预期错误",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"error", failure.Error(),
			)
		}
		response.Fail(c, failure.Code)
		c.Abort()
	}
}
