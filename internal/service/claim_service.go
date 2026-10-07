package service

import (
	"errors"
	"strings"

	"lostfound/internal/dto"
	"lostfound/internal/model"
	"lostfound/internal/repository"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
	"lostfound/pkg/timeutil"
)

// ClaimService 处理认领申请的提交、审批与取消。
type ClaimService struct {
	claims *repository.ClaimRepository
	items  *repository.ItemRepository
}

// NewClaimService 构造认领服务。
// 依赖物品仓储是因为「申请能不能提交」要读物品的类型与状态，详情还要内联物品。
func NewClaimService(claims *repository.ClaimRepository, items *repository.ItemRepository) *ClaimService {
	return &ClaimService{claims: claims, items: items}
}

// ListForAdmin 认领申请列表，仅供后台角色。
func (s *ClaimService) ListForAdmin(query dto.ClaimQuery) (dto.ClaimListResponse, error) {
	claims, err := s.claims.List(repository.ClaimFilter{
		Status: query.Status,
		Offset: query.Offset(),
		Limit:  query.PageSize,
	})
	if err != nil {
		return dto.ClaimListResponse{}, err
	}
	return dto.NewClaimListResponse(claims), nil
}

// ListMine 我的认领申请，仅本人可见。
func (s *ClaimService) ListMine(userID uint, query dto.ClaimQuery) (dto.ClaimListResponse, error) {
	query.ApplicantID = &userID

	claims, err := s.claims.List(repository.ClaimFilter{
		ApplicantID: query.ApplicantID,
		Status:      query.Status,
		Offset:      query.Offset(),
		Limit:       query.PageSize,
	})
	if err != nil {
		return dto.ClaimListResponse{}, err
	}
	return dto.NewClaimListResponse(claims), nil
}

// Submit 提交认领申请。
//
// 判定顺序按文档规定的优先级排列，不能随意调换：
// 物品不存在(4) → 不可认领(7) → 申请自己的帖子(7) → 重复申请(9) → 理由为空(1)。
// 其中「理由为空」放在最后，是为了让「对一条已关闭的帖子提交空理由」返回 7 而不是 1 ——
// 前者更能说明问题所在。
func (s *ClaimService) Submit(user *model.User, itemID uint, reason string) error {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return translate(err)
	}
	// 只有「失物招领」且「已发布」的帖子可被认领：寻物启事是失主在找东西，
	// 不存在「认领」语义；未过审或已关闭的帖子则已无认领通道。
	if item.Type != model.ItemTypeFound || item.ItemStatus != model.ItemStatusPublished {
		return apperr.New(errcode.InvalidState)
	}
	if item.PosterID == user.UserID {
		return apperr.New(errcode.InvalidState)
	}

	duplicated, err := s.claims.Exists(itemID, user.UserID)
	if err != nil {
		return err
	}
	if duplicated {
		return apperr.New(errcode.DuplicateSubmit)
	}

	if strings.TrimSpace(reason) == "" {
		return apperr.New(errcode.BadRequest)
	}

	now := timeutil.Now()
	return s.claims.Create(&model.Claim{
		ItemID:           itemID,
		Reason:           reason,
		ApplicantContact: user.Contact,
		CreatedTime:      now,
		LastEditTime:     now,
		ClaimStatus:      model.ClaimStatusPending,
		ApplicantID:      user.UserID,
	})
}

// Detail 申请详情，内联对应物品。
// 普通用户只能看自己的申请，后台角色可看全部。
func (s *ClaimService) Detail(user *model.User, claimID uint) (dto.ClaimDetailResponse, error) {
	claim, err := s.claims.FindByID(claimID)
	if err != nil {
		return dto.ClaimDetailResponse{}, translate(err)
	}
	if !claim.BelongsTo(user) && !user.IsBackOffice() {
		return dto.ClaimDetailResponse{}, apperr.New(errcode.Forbidden)
	}

	response := dto.ClaimDetailResponse{
		ClaimID:      claim.ClaimID,
		Reason:       claim.Reason,
		Contact:      claim.ApplicantContact,
		CreatedTime:  claim.CreatedTime,
		LastEditTime: claim.LastEditTime,
		ClaimStatus:  claim.ClaimStatus,
		UserID:       claim.ApplicantID,
	}

	// 关联物品可能已被删除。接口把 item 设计成可为 null，前端据此渲染「物品已不存在」，
	// 因此这里不把「物品查不到」当成错误，只有真正的查询故障才上抛。
	item, err := s.items.FindByID(claim.ItemID)
	switch {
	case err == nil:
		mapped := dto.NewItemResponse(item)
		response.Item = &mapped
	case errors.Is(err, repository.ErrNotFound):
		// 保持 Item 为 nil。
	default:
		return dto.ClaimDetailResponse{}, err
	}

	return response, nil
}

// Update 修改申请理由与联系方式，仅申请人本人。
func (s *ClaimService) Update(user *model.User, claimID uint, req dto.ClaimUpdateRequest) error {
	claim, err := s.claims.FindByID(claimID)
	if err != nil {
		return translate(err)
	}
	if !claim.BelongsTo(user) {
		return apperr.New(errcode.Forbidden)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return apperr.New(errcode.BadRequest)
	}

	claim.Reason = req.Reason
	if contact := strings.TrimSpace(req.ApplicantContact); contact != "" {
		claim.ApplicantContact = contact
	}
	claim.LastEditTime = timeutil.Now()
	return s.claims.Update(claim)
}

// Delete 取消/删除认领申请：申请人本人，或系统管理员删任意。
func (s *ClaimService) Delete(user *model.User, claimID uint) error {
	claim, err := s.claims.FindByID(claimID)
	if err != nil {
		return translate(err)
	}
	if !claim.BelongsTo(user) && !user.IsSysAdmin() {
		return apperr.New(errcode.Forbidden)
	}
	return s.claims.Delete(claimID)
}

// Review 审批认领申请。
//
// 通过是一组连带变更（本条通过 → 物品关闭 → 同物品其他待审批申请自动驳回），
// 由仓储层在同一事务内完成；驳回仅改本条状态。
func (s *ClaimService) Review(claimID uint, option string) error {
	claim, err := s.claims.FindByID(claimID)
	if err != nil {
		return translate(err)
	}
	if !claim.IsPending() {
		return apperr.New(errcode.InvalidState)
	}

	now := timeutil.Now()
	switch option {
	case "approve":
		return s.claims.Approve(claim, now)
	case "reject":
		return s.claims.UpdateStatus(claimID, model.ClaimStatusRejected, now)
	default:
		return apperr.New(errcode.BadRequest)
	}
}
