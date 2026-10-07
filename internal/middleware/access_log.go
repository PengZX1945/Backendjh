package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"lostfound/pkg/logger"
)

// AccessLog 记录每个请求的方法、路径、状态码与耗时。
//
// 注册在 ErrorHandler 之外层，这样取到的状态码已经是统一响应中间件写完之后的
// 最终值（含被翻译成 400/401/404 的业务失败），而不是默认的 200。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("请求完成",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"cost", time.Since(start).String(),
		)
	}
}
