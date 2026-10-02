package repository

import (
	"Backendjh/internal/model"
	"errors"

	"gorm.io/gorm"
)

func CreateItem(it *model.Item) error {
	return model.DB.Create(it).Error
}

func FindItemByID(id uint64) (*model.Item, error) {
	var item model.Item
	err := model.DB.Where("id = ?", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func UpdateItem(it *model.Item) error {
	return model.DB.Save(it).Error
}
func DeleteItem(id uint64) error {
	return model.DB.Delete(&model.Item{}, id).Error
}

type ItemListFilter struct {
	Type      string
	Status    *int8
	PosterID  *uint64
	Category  string
	Location  string
	Keyword   string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}

func ListItems(f ItemListFilter) ([]model.Item, error) {
	var items []model.Item

	q := model.DB.Model(&model.Item{})

	if f.Type != "" {                        // ② type 有值才加
		q = q.Where("type = ?", f.Type)
	}
	if f.Status != nil {                     // ③ status 传了才加
		q = q.Where("status = ?", *f.Status)
	}
	if f.PosterID != nil {
		q = q.Where("poster_id = ?", *f.PosterID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Location != "" {
		like := "%" + f.Location + "%"
		q = q.Where("location LIKE ?", like)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("item_name LIKE ? OR description LIKE ?", like, like)
	}
	if f.StartTime != "" {
		q = q.Where("happen_time >= ?", f.StartTime)
	}
	if f.EndTime != "" {
		q = q.Where("happen_time <= ?", f.EndTime)
	}
	q = q.Order("happen_time DESC").
		Limit(f.PageSize).
		Offset((f.Page - 1) * f.PageSize)

	err := q.Find(&items).Error
	return items, err
}
