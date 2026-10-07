package repository

import (
	"gorm.io/gorm"

	"lostfound/internal/model"
)

// AnnouncementRepository 提供公告的数据访问。
type AnnouncementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository 构造公告仓储。
func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

// Create 新增公告。
func (r *AnnouncementRepository) Create(announcement *model.Announcement) error {
	return r.db.Create(announcement).Error
}

// FindByID 按主键查询，不存在时返回 ErrNotFound。
func (r *AnnouncementRepository) FindByID(announcementID uint) (*model.Announcement, error) {
	var announcement model.Announcement
	if err := r.db.First(&announcement, "announcement_id = ?", announcementID).Error; err != nil {
		return nil, translate(err)
	}
	return &announcement, nil
}

// Update 保存标题、正文与上下线状态。
func (r *AnnouncementRepository) Update(announcement *model.Announcement) error {
	return r.db.Model(&model.Announcement{}).
		Where("announcement_id = ?", announcement.AnnouncementID).
		Updates(map[string]any{
			"title":               announcement.Title,
			"content":             announcement.Content,
			"announcement_status": announcement.AnnouncementStatus,
		}).Error
}

// Delete 删除公告。
func (r *AnnouncementRepository) Delete(announcementID uint) error {
	return r.db.Delete(&model.Announcement{}, "announcement_id = ?", announcementID).Error
}

// List 按状态分页查询，按发布时间倒序。
// status 传 nil 表示不过滤（管理端「获取所有公告」用），传具体值用于公开列表只看已公开。
func (r *AnnouncementRepository) List(status *int, offset, limit int) ([]model.Announcement, error) {
	query := r.db.Model(&model.Announcement{})
	if status != nil {
		query = query.Where("announcement_status = ?", *status)
	}

	var announcements []model.Announcement
	if err := query.Order("created_time desc").
		Limit(limit).Offset(offset).
		Find(&announcements).Error; err != nil {
		return nil, err
	}
	return announcements, nil
}
