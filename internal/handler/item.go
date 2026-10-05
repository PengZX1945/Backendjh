package handler

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"

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
	Images      []string `json:"images"`
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
