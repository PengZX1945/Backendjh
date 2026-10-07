// Package handler 是 HTTP 适配层。
//
// 职责被刻意压到最薄：解析请求（路径/查询/请求体）→ 调 service → 写响应。
// 这里不出现任何业务判断与错误码选择：失败一律 c.Error(err) 交给统一响应中间件，
// 成功一律 response.OK。因此每个 handler 都能一眼看完。
package handler

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/middleware"
	"lostfound/internal/model"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
)

// defaultPageSize 是列表默认每页条数，与前端信息流口径一致（每页 12 条）。
const defaultPageSize = 12

// maxPageSize 是分页上限，防止一次请求把整库拉走。
const maxPageSize = 100

// bindJSON 解析 JSON 请求体。
//
// 允许空请求体：文档里「通过审核」不带任何参数，此时目标结构体保持零值，
// 由 service 判断必填项缺失后返回 1。JSON 语法本身出错则直接判参数错误(1)。
func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil && !errors.Is(err, io.EOF) {
		c.Error(apperr.Wrap(errcode.BadRequest, err))
		return false
	}
	return true
}

// currentUser 取当前登录用户。路由上挂了鉴权中间件时必然非 nil；
// 万一没挂，RequireRole 会拦下，此处返回 nil 由 service 的 nil 安全判断兜住。
func currentUser(c *gin.Context) *model.User {
	return middleware.CurrentUser(c)
}

// pagination 读取分页参数：缺省或非法回落默认值，并夹在上限内。
func pagination(c *gin.Context) dto.PageQuery {
	page := intQuery(c, "page", 1)
	pageSize := intQuery(c, "page_size", defaultPageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return dto.PageQuery{Page: page, PageSize: pageSize}
}

// intQuery 读取整型查询参数，缺失或非法时返回兜底值。
func intQuery(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

// optionalIntQuery 读取可选整型查询参数。未提交或非法时返回 nil，表示「不过滤」。
func optionalIntQuery(c *gin.Context, key string) *int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &parsed
}

// textQuery 读取文本查询参数并去掉首尾空白。
func textQuery(c *gin.Context, key string) string {
	return strings.TrimSpace(c.Query(key))
}

// uintParam 读取无符号整型路径参数。非法（含非数字）时写回参数错误(1) 并返回 false。
func uintParam(c *gin.Context, key string) (uint, bool) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(c.Param(key)), 10, 64)
	if err != nil {
		c.Error(apperr.Wrap(errcode.BadRequest, err))
		return 0, false
	}
	return uint(parsed), true
}

// uintQuery 读取查询参数里的整型 ID，缺失或非法时返回 0（调用方按「未提交」处理）。
func uintQuery(c *gin.Context, key string) uint {
	parsed, err := strconv.ParseUint(strings.TrimSpace(c.Query(key)), 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}
