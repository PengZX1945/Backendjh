package service

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"lostfound/internal/config"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
)

// sniffBytes 是内容嗅探读取的头部长度，与 net/http 的建议值一致。
const sniffBytes = 512

// UploadService 处理图片上传。
type UploadService struct {
	dir         string
	urlPrefix   string
	maxBytes    int64
	allowedExts map[string]bool
}

// NewUploadService 按配置构造上传服务。
func NewUploadService(cfg config.Upload) *UploadService {
	allowed := make(map[string]bool, len(cfg.AllowedExts))
	for _, ext := range cfg.AllowedExts {
		allowed[strings.ToLower(strings.TrimSpace(ext))] = true
	}
	return &UploadService{
		dir:         cfg.Dir,
		urlPrefix:   strings.TrimRight(cfg.URLPrefix, "/"),
		maxBytes:    int64(cfg.MaxSizeMB) * 1024 * 1024,
		allowedExts: allowed,
	}
}

// Save 校验并落盘单张图片，返回可直接渲染的访问地址。
//
// 校验走两道：扩展名白名单 + 内容嗅探。只看扩展名会让改了后缀的脚本文件混进来，
// 只看内容又会误伤扩展名合法但头部不含 magic 的图片，两道都过才算数。
// 任何一项不满足都归为「文件上传失败」(8)，不细分原因 —— 对上传方而言
// 「格式不支持」与「超过大小限制」的处理动作是一样的：换一张。
func (s *UploadService) Save(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", apperr.New(errcode.UploadFailed)
	}
	if file.Size > s.maxBytes {
		return "", apperr.New(errcode.UploadFailed)
	}

	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if !s.allowedExts[extension] {
		return "", apperr.New(errcode.UploadFailed)
	}

	source, err := file.Open()
	if err != nil {
		return "", apperr.Wrap(errcode.UploadFailed, err)
	}
	defer source.Close()

	head := make([]byte, sniffBytes)
	read, err := io.ReadFull(source, head)
	// 不足 512 字节的小图会返回 ErrUnexpectedEOF / EOF，这不算失败。
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", apperr.Wrap(errcode.UploadFailed, err)
	}
	if !strings.HasPrefix(http.DetectContentType(head[:read]), "image/") {
		return "", apperr.New(errcode.UploadFailed)
	}
	// 嗅探已经消耗掉文件头，回到起点再整份写入，否则落盘的图片会缺开头 512 字节。
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", apperr.Wrap(errcode.UploadFailed, err)
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}

	// 用随机名而不是原始文件名：原始名可能重复、可能带路径分隔符或非法字符，
	// 直接落盘会成为路径穿越的入口。
	name := uuid.NewString() + "." + extension
	target, err := os.Create(filepath.Join(s.dir, name))
	if err != nil {
		return "", err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return "", err
	}
	return s.urlPrefix + "/" + name, nil
}
