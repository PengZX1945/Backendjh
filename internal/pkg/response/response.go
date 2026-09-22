package response

import (
	"Backendjh/internal/pkg/errcode"
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": data})
}

func Error(c *gin.Context, err *errcode.Error) {
	c.JSON(http.StatusOK, gin.H{"code": err.Code, "msg": err.Msg})
}
