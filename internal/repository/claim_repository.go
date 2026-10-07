package repository

import (
	"gorm.io/gorm"

	"lostfound/internal/model"
)

// ClaimFilter 是认领申请查询条件。
type ClaimFilter struct {
	ItemID      *uint
	ApplicantID *uint
	Status      *int
	Offset      int
	Limit       int
}

// ClaimRepository 提供认领申请的数据访问。
type ClaimRepository struct {
	db *gorm.DB
}

// NewClaimRepository 构造认领仓储。
func NewClaimRepository(db *gorm.DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}

// Create 新增申请。
func (r *ClaimRepository) Create(claim *model.Claim) error {
	return r.db.Create(claim).Error
}

// FindByID 按主键查询，不存在时返回 ErrNotFound。
func (r *ClaimRepository) FindByID(claimID uint) (*model.Claim, error) {
	var claim model.Claim
	if err := r.db.First(&claim, "claim_id = ?", claimID).Error; err != nil {
		return nil, translate(err)
	}
	return &claim, nil
}

// Update 保存申请人可改的字段（理由与联系方式）。
func (r *ClaimRepository) Update(claim *model.Claim) error {
	return r.db.Model(&model.Claim{}).
		Where("claim_id = ?", claim.ClaimID).
		Updates(map[string]any{
			"reason":            claim.Reason,
			"applicant_contact": claim.ApplicantContact,
			"last_edit_time":    claim.LastEditTime,
		}).Error
}

// UpdateStatus 只改状态与最后修改时间，供审批与取消使用。
func (r *ClaimRepository) UpdateStatus(claimID uint, status int, lastEditTime string) error {
	return r.db.Model(&model.Claim{}).
		Where("claim_id = ?", claimID).
		Updates(map[string]any{
			"claim_status":   status,
			"last_edit_time": lastEditTime,
		}).Error
}

// Delete 删除申请。
func (r *ClaimRepository) Delete(claimID uint) error {
	return r.db.Delete(&model.Claim{}, "claim_id = ?", claimID).Error
}

// Exists 判断同一申请人对同一物品是否已提交过申请，用于「请勿重复提交」判定。
// 已取消(3)的申请不计入：取消后应当允许重新申请。
func (r *ClaimRepository) Exists(itemID, applicantID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Claim{}).
		Where("item_id = ? AND applicant_id = ? AND claim_status <> ?", itemID, applicantID, model.ClaimStatusCanceled).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// List 按条件分页查询，按提交时间倒序。
func (r *ClaimRepository) List(filter ClaimFilter) ([]model.Claim, error) {
	query := r.db.Model(&model.Claim{})
	if filter.ItemID != nil {
		query = query.Where("item_id = ?", *filter.ItemID)
	}
	if filter.ApplicantID != nil {
		query = query.Where("applicant_id = ?", *filter.ApplicantID)
	}
	if filter.Status != nil {
		query = query.Where("claim_status = ?", *filter.Status)
	}

	var claims []model.Claim
	if err := query.Order("created_time desc").
		Limit(filter.Limit).Offset(filter.Offset).
		Find(&claims).Error; err != nil {
		return nil, err
	}
	return claims, nil
}

// Approve 在一个事务内完成「通过该申请」的全部连带变更：
//
//	① 本条申请 → 已通过
//	② 对应物品 → 已关闭（认领完成）
//	③ 同一物品的其他待审批申请 → 自动驳回
//
// 三步必须同生共死，否则会留下「物品已关闭但其他申请还挂着待审批」的中间态，
// 管理员稍后处理时会对着一个已关闭物品继续审批。故用事务封装在仓储层。
func (r *ClaimRepository) Approve(claim *model.Claim, lastEditTime string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Claim{}).
			Where("claim_id = ?", claim.ClaimID).
			Updates(map[string]any{
				"claim_status":   model.ClaimStatusApproved,
				"last_edit_time": lastEditTime,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Item{}).
			Where("item_id = ?", claim.ItemID).
			Updates(map[string]any{
				"item_status":    model.ItemStatusClosed,
				"last_edit_time": lastEditTime,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&model.Claim{}).
			Where("item_id = ? AND claim_id <> ? AND claim_status = ?",
				claim.ItemID, claim.ClaimID, model.ClaimStatusPending).
			Updates(map[string]any{
				"claim_status":   model.ClaimStatusRejected,
				"last_edit_time": lastEditTime,
			}).Error
	})
}
