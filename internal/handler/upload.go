package handler

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

func (uh *UploadHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, errcode.UploadFailed)
		return
	}
	url, e := uh.uploadService.SaveImage(file)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, gin.H{"url": url})
}
