package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/repository"
)

type ClaimService struct{}

func NewClaimService() *ClaimService {
	return &ClaimService{}
}

func (s *ClaimService) Submit(userID, itemID uint64, reason string) *errcode.Error {
	if reason == "" {
		return errcode.ParamError
	}
	item, err := repository.FindItemByID(itemID)
	if err != nil {
		return errcode.InternalError
	}
	if item == nil {
		return errcode.NotFound
	}
	if item.Type != model.ItemTypeFound || item.Status != model.ItemStatusPublished {
		return errcode.StatusNotAllowed
	}
	if item.PosterID == userID {
		return errcode.StatusNotAllowed
	}
	n, err := repository.CountActiveClaim(itemID, userID)
	if err != nil {
		return errcode.InternalError
	}
	if n > 0 {
		return errcode.DuplicateSubmit
	}
	u, err := repository.FindUserByID(userID)
	if err != nil {
		return errcode.InternalError
	}
	if u == nil {
		return errcode.Unauthorized
	}
	claim := &model.Claim{
		ItemID:           itemID,
		ApplicantID:      userID,
		Reason:           reason,
		ApplicantContact: u.Contact,
		ClaimStatus:      model.ClaimStatusPending,
	}
	if err := repository.CreateClaim(claim); err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *ClaimService) Review(role string, claimID uint64, option string) *errcode.Error {
	if role != model.RoleFinderAdmin && role != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	if option != "approve" && option != "reject" {
		return errcode.ParamError
	}
	claim, err := repository.FindClaimByID(claimID)
	if err != nil {
		return errcode.InternalError
	}
	if claim == nil {
		return errcode.NotFound
	}
	if claim.ClaimStatus != model.ClaimStatusPending {
		return errcode.StatusNotAllowed
	}
	if option == "approve" {
		claim.ClaimStatus = model.ClaimStatusApproved
		if err := repository.UpdateClaim(claim); err != nil {
			return errcode.InternalError
		}
		item, err := repository.FindItemByID(claim.ItemID)
		if err != nil {
			return errcode.InternalError
		}
		if item != nil {
			item.Status = model.ItemStatusClaimed
			if err := repository.UpdateItem(item); err != nil {
				return errcode.InternalError
			}
		}
		if err := repository.RejectOtherPendingClaims(claim.ItemID, claim.ID); err != nil {
			return errcode.InternalError
		}
		return nil
	}
	claim.ClaimStatus = model.ClaimStatusRejected
	if err := repository.UpdateClaim(claim); err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *ClaimService) Delete(userID uint64, role string, claimID uint64) *errcode.Error {
	if claimID == 0 {
		return errcode.ParamError
	}
	claim, err := repository.FindClaimByID(claimID)
	if err != nil {
		return errcode.InternalError
	}
	if claim == nil {
		return errcode.NotFound
	}
	if claim.ApplicantID != userID && role != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	err = repository.DeleteClaim(claim.ID)
	if err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *ClaimService) MyClaims(userID uint64, claimStatus *int8, page int, pageSize int) ([]model.Claim, *errcode.Error) {
	if claimStatus != nil && (*claimStatus < 0 || *claimStatus > model.ClaimStatusRejected) {
		return nil, errcode.ParamError
	}
	claimFilter := repository.ClaimListFilter{ApplicantID: &userID, ClaimStatus: claimStatus, Page: page, PageSize: pageSize}
	claimList, err := repository.ListClaims(claimFilter)
	if err != nil {
		return nil, errcode.InternalError
	}
	return claimList, nil
}

func (s *ClaimService) Detail(userID uint64, role string, claimID uint64) (*model.Claim, *errcode.Error) {
	claim, err := repository.FindClaimByID(claimID)
	if err != nil {
		return nil, errcode.InternalError
	}
	if claim == nil {
		return nil, errcode.NotFound
	}
	if userID != 0 && claim.ApplicantID == userID || role == model.RoleSysAdmin || role == model.RoleFinderAdmin {
		return claim, nil
	}
	return nil, errcode.NotFound
}
