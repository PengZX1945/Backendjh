package handler

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct {
	svc *service.AnnouncementService
}

func NewAnnouncementHandler(s *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: s}
}

// List 公开：所有人可读最新公告
func (h *AnnouncementHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	list, e := h.svc.List(limit)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, gin.H{"announcements": list})
}

type upsertAnnouncementRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req upsertAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	a, e := h.svc.Create(c.GetString("role"), req.Title, req.Content, c.GetUint64("userID"))
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, gin.H{"announcement": a})
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("announcement_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req upsertAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	e := h.svc.Update(c.GetString("role"), id, req.Title, req.Content)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("announcement_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	e := h.svc.Delete(c.GetString("role"), id)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}
