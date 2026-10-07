package handler

import (
	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/service"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
	"lostfound/pkg/response"
)

// UploadHandler 处理图片上传。
type UploadHandler struct {
	uploads *service.UploadService
}

// NewUploadHandler 构造上传处理器。
func NewUploadHandler(uploads *service.UploadService) *UploadHandler {
	return &UploadHandler{uploads: uploads}
}

// Upload 处理 POST /api/upload（multipart/form-data，字段名 file）。
// 任何失败（缺字段、格式不支持、超限）都归为错误码 8。
func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(apperr.Wrap(errcode.UploadFailed, err))
		return
	}

	url, err := h.uploads.Save(file)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, dto.UploadResponse{URL: url})
}
