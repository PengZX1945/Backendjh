package middleware

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/jwt"
	"Backendjh/internal/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenStr, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || tokenStr == "" {
			response.Fail(c, errcode.Unauthorized)
			c.Abort()
			return
		}
		claims, err := jwt.Parse(secret, tokenStr)
		if err != nil {
			response.Fail(c, errcode.Unauthorized)
			c.Abort()
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
