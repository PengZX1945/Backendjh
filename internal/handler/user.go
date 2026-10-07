package handler

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(u *service.UserService) *UserHandler {
	return &UserHandler{userService: u}
}

func userResponse(u *model.User) gin.H {
	return gin.H{
		"user_id":  u.ID,
		"username": u.Username,
		"nickname": u.Nickname,
		"role":     u.Role,
		"contact":  u.Contact,
		"status":   u.Status,
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	token, _, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"token": token})
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
	Contact  string `json:"contact" binding:"required"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	_, err := h.userService.Register(req.Username, req.Password, req.Nickname, req.Contact)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint64("userID")
	u, e := h.userService.GetProfile(userID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"nickname": u.Nickname,
		"role":     u.Role,
		"contact":  u.Contact,
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) Logout(c *gin.Context) {
	response.OK(c, nil)
}

type updateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Contact  string `json:"contact" binding:"required"`
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	_, err := h.userService.UpdateProfile(userID, req.Nickname, req.Contact)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) AdminUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	users, e := h.userService.AdminUserList(
		c.GetString("role"),
		c.Query("keyword"),
		c.Query("role"),
		page, pageSize,
	)
	if e != nil {
		response.Fail(c, e)
		return
	}
	list := make([]gin.H, 0, len(users))
	for i := range users {
		list = append(list, userResponse(&users[i]))
	}
	response.OK(c, gin.H{"users": list})
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	e := h.userService.UpdateRole(userID, role, targetID, req.Role)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}

type updateStatusRequest struct {
	Status int8 `json:"status"`
}

func (h *UserHandler) UpdateStatus(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	e := h.userService.UpdateStatus(userID, role, targetID, req.Status)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}
