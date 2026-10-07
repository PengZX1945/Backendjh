package handler

import (
	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/service"
	"lostfound/pkg/response"
)

// AuthHandler 处理注册、登录、个人资料与用户管理接口。
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler 构造鉴权处理器。
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register 处理 POST /api/auth/register。
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.Register(req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Login 处理 POST /api/auth/login，凭证放在 data.token。
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	session, err := h.auth.Login(req)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, session)
}

// Logout 处理 POST /api/auth/logout。
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.auth.Logout(); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Profile 处理 GET /api/auth/profile。
func (h *AuthHandler) Profile(c *gin.Context) {
	profile, err := h.auth.Profile(currentUser(c).UserID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, profile)
}

// UpdateProfile 处理 PUT /api/auth/profile?user_id=。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if !bindJSON(c, &req) {
		return
	}
	user := currentUser(c)
	if err := h.auth.UpdateProfile(user.UserID, uintQuery(c, "user_id"), req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// ChangePassword 处理 PUT /api/auth/password?user_id=。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	user := currentUser(c)
	if err := h.auth.ChangePassword(user.UserID, uintQuery(c, "user_id"), req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// ListUsers 处理 GET /api/admin/users（系统管理员）。
func (h *AuthHandler) ListUsers(c *gin.Context) {
	result, err := h.auth.ListUsers(dto.UserQuery{
		PageQuery: pagination(c),
		Role:      textQuery(c, "role"),
		Keyword:   textQuery(c, "keyword"),
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// UpdateUserRole 处理 PUT /api/admin/users/{user_id}/role（系统管理员）。
func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	targetUserID, ok := uintParam(c, "user_id")
	if !ok {
		return
	}
	var req dto.UpdateUserRoleRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.UpdateUserRole(currentUser(c).UserID, targetUserID, req.Role); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}
