package handler

import (
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
	response.OK(c, gin.H{
		"item_id":        item.ID,
		"type":           item.Type,
		"item_name":      item.ItemName,
		"category":       item.Category,
		"location":       item.Location,
		"happen_time":    item.HappenTime,
		"description":    item.Description,
		"image":          item.GetImages(),
		"item_status":    item.Status,
		"reject_reason":  item.RejectReason,
		"poster_id":      item.PosterID,
		"get_contact":    item.GetContact,
		"get_location":   item.GetLocation,
		"created_time":   item.CreatedAt,
		"last_edit_time": item.UpdatedAt,
	})
	return
}
