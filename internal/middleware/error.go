package middleware

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				response.Fail(c, errcode.InternalError)
				c.Abort()
			}
		}()
		c.Next()
	}
}
