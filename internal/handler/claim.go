package handler

import (
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/response"
	"Backendjh/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClaimHandler struct {
	claimService *service.ClaimService
}

func NewClaimHandler(s *service.ClaimService) *ClaimHandler {
	return &ClaimHandler{claimService: s}
}

type submitRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (ch *ClaimHandler) Submit(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Query("item_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	e := ch.claimService.Submit(userID, itemID, req.Reason)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
}
