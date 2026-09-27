package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		//部署时待使用
		/*
			origin := c.GetHeader("Origin")
			allowed := map[string]bool{
				"http://your-domain.com": true,
				"https://your-domain.com": true,
			}
			if allowed[origin] {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		*/
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
