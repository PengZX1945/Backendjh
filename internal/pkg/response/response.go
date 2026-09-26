package response

import (
	"Backendjh/internal/pkg/errcode"
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": data})
}

func Fail(c *gin.Context, err *errcode.Error) {
	c.JSON(httpStatusOf(err), gin.H{"code": err.Code, "msg": err.Msg})
}

func httpStatusOf(err *errcode.Error) int {
	switch err {
	case errcode.SamePassword:
		return http.StatusBadRequest
	case errcode.ParamError:
		return http.StatusBadRequest // 400
	case errcode.Unauthorized:
		return http.StatusUnauthorized // 401
	case errcode.Forbidden:
		return http.StatusForbidden // 403
	case errcode.NotFound:
		return http.StatusNotFound // 404
	case errcode.StatusNotAllowed:
		return http.StatusMethodNotAllowed // 405
	case errcode.UploadFailed:
		return http.StatusNotAcceptable // 406
	case errcode.UsernameExists, errcode.DuplicateSubmit:
		return http.StatusConflict // 409
	case errcode.AccountDisabled:
		return http.StatusLocked // 423
	case errcode.InternalError:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusOK // 200
	}
}
