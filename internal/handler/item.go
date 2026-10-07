package handler

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/repository"
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
	itemID, err := strconv.ParseUint(c.Param("type"), 10, 64)
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
		Typ:       c.Param("type"),
		Category:  c.Query("category"),
		Location:  c.Query("location"),
		Keyword:   c.Query("keyword"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Page:      page,
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

func (ih *ItemHandler) MyItem(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	typ := c.Query("type")
	var itemStatus *int8
	if s := c.Query("item_status"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			response.Fail(c, errcode.ParamError)
			return
		}
		status := int8(v)
		itemStatus = &status
	}
	userID := c.GetUint64("userID")
	myList, e := ih.itemService.MyItem(userID, typ, itemStatus, page, pageSize)
	if e != nil {
		response.Fail(c, e)
		return
	}
	items := make([]gin.H, 0, len(myList))
	for i := range myList {
		items = append(items, itemResponse(&myList[i]))
	}
	response.OK(c, gin.H{"items": items})
	return
}

type updateRequest struct {
	Type        string   `json:"type" binding:"required"`
	ItemName    string   `json:"item_name" binding:"required"`
	Category    string   `json:"category" binding:"required"`
	Location    string   `json:"location"`
	HappenTime  string   `json:"happen_time"`
	Description string   `json:"description"`
	Images      []string `json:"image"`
	GetContact  string   `json:"get_contact"`
	GetLocation string   `json:"get_location"`
}

func (ih *ItemHandler) Update(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	e := ih.itemService.Update(userID, itemID, service.UpdateParams{
		Typ:         req.Type,
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
	return
}

func (ih *ItemHandler) PendingList(c *gin.Context) {
	typ := c.Param("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	role := c.GetString("role")
	items, e := ih.itemService.PendingList(role, typ, page, pageSize)
	if e != nil {
		response.Fail(c, e)
		return
	}
	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, itemResponse(&items[i]))
	}
	response.OK(c, gin.H{"items": list})
}

type reviewRequest struct {
	RejectReason string `json:"reject_reason"`
}

func (ih *ItemHandler) Review(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	option := c.Param("option")
	var req reviewRequest
	_ = c.ShouldBindJSON(&req)
	role := c.GetString("role")
	e := ih.itemService.Review(role, itemID, option, req.RejectReason)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}

func (ih *ItemHandler) AdminClose(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	role := c.GetString("role")
	e := ih.itemService.AdminClose(role, itemID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}

func (ih *ItemHandler) AdminItemList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	var itemStatus *int8
	if s := c.Query("status"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			response.Fail(c, errcode.ParamError)
			return
		}
		status := int8(v)
		itemStatus = &status
	}
	role := c.GetString("role")
	items, e := ih.itemService.AdminItemList(role, repository.ItemListFilter{
		Type:     c.Query("type"),
		Category: c.Query("category"),
		Keyword:  c.Query("keyword"),
		Status:   itemStatus,
		Page:     page,
		PageSize: pageSize,
	})
	if e != nil {
		response.Fail(c, e)
		return
	}
	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, itemResponse(&items[i]))
	}
	response.OK(c, gin.H{"items": list})
}