package model

import "time"

type Claim struct {
	ID               uint64    `gorm:"primaryKey" json:"claim_id"`
	ItemID           uint64    `gorm:"not null;index" json:"item_id"`
	ApplicantID      uint64    `gorm:"not null;index" json:"applicant_id"`
	Reason           string    `gorm:"size:1024" json:"reason"`
	ApplicantContact string    `gorm:"size:64" json:"applicant_contact"`
	ClaimStatus      int8      `gorm:"not null;default:0;index" json:"claim_status"`
	CreatedAt        time.Time `json:"created_time"`
	UpdatedAt        time.Time `json:"last_edit_time"`
}

const (
	ClaimStatusPending  int8 = 0 // 待审批
	ClaimStatusApproved int8 = 1 // 已通过
	ClaimStatusRejected int8 = 2 // 已驳回
)
