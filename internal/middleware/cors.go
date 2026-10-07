package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 允许跨域访问。
//
// 开发态前端经 vite 代理访问后端本是同源，本中间件主要服务两种场景：绕过代理直连
// 后端调试、以及把前端产物放到别的端口访问。预检请求（OPTIONS）直接以 204 结束，
// 不进业务处理链，避免被鉴权中间件拦成 401。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			// 非浏览器请求（curl、服务间调用）没有 Origin，给通配值即可。
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")
		// 响应随 Origin 变化，提示缓存按 Origin 分桶，避免被中间缓存串用。
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
