package service

import (
	"Backendjh/internal/pkg/errcode"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedImagineTypes = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
}

const maxUploadSize = 5 << 20

type UploadService struct{}

func NewUploadService() *UploadService {
	return &UploadService{}
}

func (s *UploadService) SaveImage(file *multipart.FileHeader) (string, *errcode.Error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if _, ok := allowedImagineTypes[ext]; !ok {
		return "", errcode.UploadFailed
	}
	if file.Size > maxUploadSize {
		return "", errcode.UploadFailed
	}
	newName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	src, err := file.Open()
	if err != nil {
		return "", errcode.UploadFailed
	}
	defer src.Close()
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", errcode.InternalError
	}
	dst, err := os.Create(filepath.Join("uploads", newName))
	if err != nil {
		return "", errcode.InternalError
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", errcode.InternalError
	}
	return "/uploads/" + newName, nil
}
