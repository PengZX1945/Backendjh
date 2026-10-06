package handler

import (
	"Backendjh/internal/model"
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

func claimResponse(cl *model.Claim) gin.H {
	return gin.H{
		"claim_id":          cl.ID,
		"item_id":           cl.ItemID,
		"reason":            cl.Reason,
		"applicant_contact": cl.ApplicantContact,
		"claim_status":      cl.ClaimStatus,
		"applicant_id":      cl.ApplicantID,
		"created_time":      cl.CreatedAt,
		"last_edit_time":    cl.UpdatedAt,
	}
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

func (ch *ClaimHandler) Delete(c *gin.Context) {
	claimID, err := strconv.ParseUint(c.Param("claim_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	e := ch.claimService.Delete(userID, role, claimID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
	return
}

func (ch *ClaimHandler) Detail(c *gin.Context) {
	claimID, err := strconv.ParseUint(c.Param("claim_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	role := c.GetString("role")
	claim, e := ch.claimService.Detail(userID, role, claimID)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, claimResponse(claim))
	return
}

type updateClaimRequest struct {
	Reason           string `json:"reason" binding:"required"`
	ApplicantContact string `json:"applicant_contact" binding:"required"`
}

func (ch *ClaimHandler) Update(c *gin.Context) {
	claimID, err := strconv.ParseUint(c.Param("claim_id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	var req updateClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ParamError)
		return
	}
	userID := c.GetUint64("userID")
	e := ch.claimService.Update(userID, claimID, req.Reason, req.ApplicantContact)
	if e != nil {
		response.Fail(c, e)
		return
	}
	response.OK(c, nil)
	return
}

func (ch *ClaimHandler) MyClaims(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	var claimStatus *int8
	if s := c.Query("claim_status"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			response.Fail(c, errcode.ParamError)
			return
		}
		status := int8(v)
		claimStatus = &status
	}
	userID := c.GetUint64("userID")
	myList, e := ch.claimService.MyClaims(userID, claimStatus, page, pageSize)
	if e != nil {
		response.Fail(c, e)
		return
	}
	claims := make([]gin.H, 0, len(myList))
	for i := range myList {
		claims = append(claims, claimResponse(&myList[i]))
	}
	response.OK(c, gin.H{"claims": claims})
	return
}
