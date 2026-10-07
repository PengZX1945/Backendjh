package repository

import (
	"Backendjh/internal/model"

	"gorm.io/gorm"
)

func ListAnnouncements(limit int) ([]model.Announcement, error) {
	var list []model.Announcement
	q := model.DB.Model(&model.Announcement{}).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}

func FindAnnouncementByID(id uint64) (*model.Announcement, error) {
	var a model.Announcement
	err := model.DB.Where("id = ?", id).First(&a).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func CreateAnnouncement(a *model.Announcement) error {
	return model.DB.Create(a).Error
}

func UpdateAnnouncement(a *model.Announcement) error {
	return model.DB.Save(a).Error
}

func DeleteAnnouncement(id uint64) error {
	return model.DB.Delete(&model.Announcement{}, "id = ?", id).Error
}
