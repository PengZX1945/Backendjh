package repository

import (
	"errors"

	"gorm.io/gorm"

	"Backendjh/internal/model"
)

func FindUserByUsername(username string) (*model.User, error) {
	var u model.User
	err := model.DB.Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func FindUserByID(id uint64) (*model.User, error) {
	var u model.User
	err := model.DB.First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func CreateUser(u *model.User) error { return model.DB.Create(u).Error }

func UpdateUser(u *model.User) error { return model.DB.Save(u).Error }

type UserListFilter struct {
	Keyword        string
	Role           string
	Page, PageSize int
}

func ListUsers(f UserListFilter) ([]model.User, error) {
	var users []model.User
	q := model.DB.Model(&model.User{})
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}
	if f.Role != "" {
		q = q.Where("role = ?", f.Role)
	}
	err := q.Order("created_at DESC").
		Limit(f.PageSize).
		Offset((f.Page - 1) * f.PageSize).
		Find(&users).Error
	return users, err
}
