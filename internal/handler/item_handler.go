package handler

import (
	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/service"
	"lostfound/pkg/response"
)

// ItemHandler 处理物品的公开查询、发布修改与审核接口。
type ItemHandler struct {
	items *service.ItemService
}

// NewItemHandler 构造物品处理器。
func NewItemHandler(items *service.ItemService) *ItemHandler {
	return &ItemHandler{items: items}
}

// List 处理 GET /api/items/list/{type}/，公开信息流。
// itemType 由路由层固定为 lost / found 注入 —— 与发布接口同一种写法，
// 让「大类只能是这两个值」成为路由层的硬约束，不必在函数体里再校验一次。
func (h *ItemHandler) List(itemType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := h.items.ListPublished(itemType, dto.ItemQuery{
			PageQuery: pagination(c),
			Category:  textQuery(c, "category"),
			Keyword:   textQuery(c, "keyword"),
			Location:  textQuery(c, "location"),
			StartTime: textQuery(c, "start_time"),
			EndTime:   textQuery(c, "end_time"),
		})
		if err != nil {
			c.Error(err)
			return
		}
		response.OK(c, result)
	}
}

// Detail 处理 GET /api/items/{item_id}。
// 挂了可选鉴权：游客也能看，但待审核/已驳回的帖子要靠访问者身份决定是否可见。
func (h *ItemHandler) Detail(c *gin.Context) {
	itemID, ok := uintParam(c, "item_id")
	if !ok {
		return
	}
	result, err := h.items.Detail(currentUser(c), itemID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Create 处理 POST /api/items/{type}。itemType 由路由层固定为 lost / found 注入。
func (h *ItemHandler) Create(itemType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ItemRequest
		if !bindJSON(c, &req) {
			return
		}
		if err := h.items.Create(currentUser(c), itemType, req); err != nil {
			c.Error(err)
			return
		}
		response.OK(c, nil)
	}
}

// Update 处理 PUT /api/items/{item_id}，全段更新。
func (h *ItemHandler) Update(c *gin.Context) {
	itemID, ok := uintParam(c, "item_id")
	if !ok {
		return
	}
	var req dto.ItemRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.items.Update(currentUser(c), itemID, req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Delete 处理 DELETE /api/items/{item_id}。
func (h *ItemHandler) Delete(c *gin.Context) {
	itemID, ok := uintParam(c, "item_id")
	if !ok {
		return
	}
	if err := h.items.Delete(currentUser(c), itemID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// ListMine 处理 GET /api/my/items。
func (h *ItemHandler) ListMine(c *gin.Context) {
	result, err := h.items.ListMine(currentUser(c).UserID, dto.ItemQuery{
		PageQuery: pagination(c),
		Type:      textQuery(c, "type"),
		Status:    optionalIntQuery(c, "item_status"),
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// CloseClaim 处理 POST /api/items/{item_id}/close，关闭认领通道。
func (h *ItemHandler) CloseClaim(c *gin.Context) {
	itemID, ok := uintParam(c, "item_id")
	if !ok {
		return
	}
	if err := h.items.CloseClaimChannel(currentUser(c), itemID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// ListPending 处理 GET /api/admin/items/pending/{type}/，审核台队列。
func (h *ItemHandler) ListPending(itemType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := h.items.ListPending(itemType, dto.ItemQuery{
			PageQuery: pagination(c),
		})
		if err != nil {
			c.Error(err)
			return
		}
		response.OK(c, result)
	}
}

// ListAll 处理 GET /api/admin/items/，全校总览（系统管理员）。
func (h *ItemHandler) ListAll(c *gin.Context) {
	result, err := h.items.ListAll(dto.ItemQuery{
		PageQuery: pagination(c),
		Type:      textQuery(c, "type"),
		Status:    optionalIntQuery(c, "status"),
		Category:  textQuery(c, "category"),
		Keyword:   textQuery(c, "keyword"),
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Review 处理 POST /api/admin/items/{item_id}/{option}。
// option（approve / reject）由路由层从两条静态路径注入，未知取值根本不会命中路由。
func (h *ItemHandler) Review(option string) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, ok := uintParam(c, "item_id")
		if !ok {
			return
		}
		var req dto.ItemReviewRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := h.items.Review(itemID, option, req.RejectReason)
		if err != nil {
			c.Error(err)
			return
		}
		response.OK(c, result)
	}
}

// SetStatus 处理 PUT /api/admin/items/{item_id}/，仅支持置为已关闭。
func (h *ItemHandler) SetStatus(c *gin.Context) {
	itemID, ok := uintParam(c, "item_id")
	if !ok {
		return
	}
	var req dto.SetItemStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.items.SetStatusClosed(itemID, req.Status); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}
