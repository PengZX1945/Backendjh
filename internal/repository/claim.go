package repository

import (
	"Backendjh/internal/model"
	"errors"

	"gorm.io/gorm"
)

func CreateClaim(c *model.Claim) error {
	return model.DB.Create(c).Error
}

func FindClaimByID(id uint64) (*model.Claim, error) {
	var claim model.Claim
	err := model.DB.Where("id = ?", id).First(&claim).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &claim, err
}

func UpdateClaim(c *model.Claim) error {
	return model.DB.Save(c).Error
}

func DeleteClaim(id uint64) error {
	return model.DB.Where("id = ?", id).Delete(&model.Claim{}).Error
}

func CountActiveClaim(itemID, applicantID uint64) (int64, error) {
	var n int64
	err := model.DB.Model(&model.Claim{}).
		Where("item_id = ? AND applicant_id = ? AND claim_status IN ?",
			itemID, applicantID, []int8{model.ClaimStatusPending, model.ClaimStatusApproved}).
		Count(&n).Error
	return n, err
}

type ClaimListFilter struct {
	ApplicantID    *uint64 // 我的列表必传；管理端传 nil = 不限
	ClaimStatus    *int8   // 空 = 不限
	Page, PageSize int
}

func ListClaims(f ClaimListFilter) ([]model.Claim, error) {
	var claims []model.Claim
	q := model.DB.Model(&model.Claim{})
	if f.ApplicantID != nil {
		q = q.Where("applicant_id = ?", *f.ApplicantID)
	}
	if f.ClaimStatus != nil {
		q = q.Where("claim_status = ?", *f.ClaimStatus)
	}
	err := q.Order("created_at DESC").
		Limit(f.PageSize).
		Offset((f.Page - 1) * f.PageSize).
		Find(&claims).Error
	return claims, err
}

func RejectOtherPendingClaims(itemID, exceptID uint64) error {
	return model.DB.Model(&model.Claim{}).
		Where("item_id = ? AND id <> ? AND claim_status = ?",
			itemID, exceptID, model.ClaimStatusPending).
		Update("claim_status", model.ClaimStatusRejected).Error
}
