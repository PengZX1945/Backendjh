package handler

import (
	"github.com/gin-gonic/gin"

	"lostfound/internal/dto"
	"lostfound/internal/service"
	"lostfound/pkg/response"
)

// ClaimHandler 处理认领申请的提交、查询与审批接口。
type ClaimHandler struct {
	claims *service.ClaimService
}

// NewClaimHandler 构造认领处理器。
func NewClaimHandler(claims *service.ClaimService) *ClaimHandler {
	return &ClaimHandler{claims: claims}
}

// ListForAdmin 处理 GET /api/admin/claims（后台角色）。
func (h *ClaimHandler) ListForAdmin(c *gin.Context) {
	result, err := h.claims.ListForAdmin(dto.ClaimQuery{
		PageQuery: pagination(c),
		Status:    optionalIntQuery(c, "claim_status"),
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// ListMine 处理 GET /api/my/claims/。
func (h *ClaimHandler) ListMine(c *gin.Context) {
	result, err := h.claims.ListMine(currentUser(c).UserID, dto.ClaimQuery{
		PageQuery: pagination(c),
		Status:    optionalIntQuery(c, "claim_status"),
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Submit 处理 POST /api/claims?item_id=。
func (h *ClaimHandler) Submit(c *gin.Context) {
	var req dto.ClaimSubmitRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.claims.Submit(currentUser(c), uintQuery(c, "item_id"), req.Reason); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Detail 处理 GET /api/claims/{claim_id}。
func (h *ClaimHandler) Detail(c *gin.Context) {
	claimID, ok := uintParam(c, "claim_id")
	if !ok {
		return
	}
	result, err := h.claims.Detail(currentUser(c), claimID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

// Update 处理 PUT /api/claims/{claim_id}。
func (h *ClaimHandler) Update(c *gin.Context) {
	claimID, ok := uintParam(c, "claim_id")
	if !ok {
		return
	}
	var req dto.ClaimUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.claims.Update(currentUser(c), claimID, req); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Delete 处理 DELETE /api/claims/{claim_id}/。
func (h *ClaimHandler) Delete(c *gin.Context) {
	claimID, ok := uintParam(c, "claim_id")
	if !ok {
		return
	}
	if err := h.claims.Delete(currentUser(c), claimID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, nil)
}

// Review 处理 POST /api/admin/claims/{claim_id}/{option}。
// option（approve / reject）由路由层从两条静态路径注入。
func (h *ClaimHandler) Review(option string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimID, ok := uintParam(c, "claim_id")
		if !ok {
			return
		}
		if err := h.claims.Review(claimID, option); err != nil {
			c.Error(err)
			return
		}
		response.OK(c, nil)
	}
}
