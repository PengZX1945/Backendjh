package handler

import (
	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/service"
	"lostfound/pkg/response"
)

// AnnouncementHandler 处理公告的公开查询与管理端维护接口。
type AnnouncementHandler struct {
	announcements *service.AnnouncementService
}

// NewAnnouncementHandler 构造公告处理器。
func NewAnnouncementHandler(announcements *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{announcements: announcements}
}

// ListPublic 处理 GET /api/announcements/。
func (h *AnnouncementHandler) ListPublic(c *gin.Context) {
	result, err := h.announcements.ListPublic(dto.AnnouncementQuery{PageQuery: pagination(c)})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Detail 处理 GET /api/announcements/{announcement_id}。
func (h *AnnouncementHandler) Detail(c *gin.Context) {
	announcementID, ok := uintParam(c, "announcement_id")
	if !ok {
		return
	}
	result, err := h.announcements.Detail(announcementID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// ListAll 处理 GET /api/admin/announcements/（系统管理员，含已下线）。
func (h *AnnouncementHandler) ListAll(c *gin.Context) {
	result, err := h.announcements.ListAll(dto.AnnouncementQuery{PageQuery: pagination(c)})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Create 处理 POST /api/admin/announcements/（系统管理员）。
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req dto.AnnouncementCreateRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.announcements.Create(req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Update 处理 PUT /api/admin/announcements/{announcement_id}（系统管理员）。
// 同一接口兼作上下线：请求体里带 announcement_status 即可。
func (h *AnnouncementHandler) Update(c *gin.Context) {
	announcementID, ok := uintParam(c, "announcement_id")
	if !ok {
		return
	}
	var req dto.AnnouncementUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.announcements.Update(announcementID, req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Delete 处理 DELETE /api/admin/announcements/{announcement_id}（系统管理员）。
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	announcementID, ok := uintParam(c, "announcement_id")
	if !ok {
		return
	}
	if err := h.announcements.Delete(announcementID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}
