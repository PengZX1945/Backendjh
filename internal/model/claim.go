package model

// 认领申请状态机：提交落 0，审批通过落 1、驳回落 2，其他同物品申请被自动驳回落 2，
// 申请人自行取消落 3。
const (
	ClaimStatusPending  = 0 // 待审批
	ClaimStatusApproved = 1 // 已通过
	ClaimStatusRejected = 2 // 已驳回
	ClaimStatusCanceled = 3 // 已取消
)

// Claim 是对某条「失物招领」帖子发起的认领申请。
type Claim struct {
	ClaimID uint `gorm:"primaryKey;column:claim_id" json:"claim_id"`
	ItemID  uint `gorm:"not null;index" json:"item_id"`
	// Reason 申请理由，用于管理员核对物品特征。
	Reason string `gorm:"size:512;not null" json:"reason"`
	// ApplicantContact 申请人联系方式快照（同样刻意冗余，便于留痕）。
	ApplicantContact string `gorm:"size:128" json:"applicant_contact"`
	CreatedTime      string `gorm:"size:20;not null;index" json:"created_time"`
	LastEditTime     string `gorm:"size:20;not null" json:"last_edit_time"`
	ClaimStatus      int    `gorm:"not null;default:0;index" json:"claim_status"`
	ApplicantID      uint   `gorm:"not null;index" json:"applicant_id"`
}

// IsPending 判断申请是否仍处于待审批状态。
func (c *Claim) IsPending() bool {
	return c.ClaimStatus == ClaimStatusPending
}

// BelongsTo 判断该申请是否由指定用户发起（申请人本人可改理由、可取消）。
func (c *Claim) BelongsTo(user *User) bool {
	return user != nil && c.ApplicantID == user.UserID
}
