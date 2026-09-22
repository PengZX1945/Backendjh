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
