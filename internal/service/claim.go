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
