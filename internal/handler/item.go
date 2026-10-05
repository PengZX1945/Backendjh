package handler

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ItemHandler struct {
	itemService *service.ItemService
}

func NewItemHandler(s *service.ItemService) *ItemHandler {
	return &ItemHandler{itemService: s}
}

func itemResponse(it *model.Item) gin.H {
	return gin.H{
		"item_id":        it.ID,
		"type":           it.Type,
		"item_name":      it.ItemName,
		"category":       it.Category,
		"location":       it.Location,
		"happen_time":    it.HappenTime,
		"description":    it.Description,
		"image":          it.GetImages(),
		"item_status":    it.Status,
		"reject_reason":  it.RejectReason,
		"poster_id":      it.PosterID,
		"get_contact":    it.GetContact,
		"get_location":   it.GetLocation,
		"created_time":   it.CreatedAt,
		"last_edit_time": it.UpdatedAt,
	}
}

type publishRequest struct {
	ItemName    string   `json:"item_name" binding:"required"`
	Category    string   `json:"category" binding:"required"`
	Location    string   `json:"location"`
	HappenTime  string   `json:"happen_time"`
	Description string   `json:"description"`
	Images      []string `json:"image"`
	GetContact  string   `json:"get_contact"`
	GetLocation string   `json:"get_location"`
}

func (ih *ItemHandler) Publish(c *gin.Context) {
	typ := c.Param("type")
	var req publishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	e := ih.itemService.Publish(userID, typ, service.PublishParams{
		ItemName:    req.ItemName,
		Category:    req.Category,
		Location:    req.Location,
		HappenTime:  req.HappenTime,
		Description: req.Description,
		Images:      req.Images,
		GetContact:  req.GetContact,
		GetLocation: req.GetLocation,
	})
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}

func (ih *ItemHandler) Close(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	e := ih.itemService.Close(userID, itemID, role)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
	return
}

func (ih *ItemHandler) Delete(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	e := ih.itemService.Delete(userID, role, itemID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
	return
}

func (ih *ItemHandler) Detail(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	item, e := ih.itemService.Detail(userID, role, itemID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, itemResponse(item))
	return
}

func (ih *ItemHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	var listParam = service.ListParams{
		Typ:       c.Param("type"),     // 路径参数：/api/items/list/:type
		Category:  c.Query("category"), // 没传就是空串 → 不过滤
		Location:  c.Query("location"),
		Keyword:   c.Query("keyword"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Page:      page, // 用你上面清洗过的变量
		PageSize:  pageSize,
	}
	list, e := ih.itemService.ListItems(listParam)
	if e != nil {
		response.Fail(c, e)
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, itemResponse(&list[i]))
	}
	response.OK(c, gin.H{"items": items})
	return
}
