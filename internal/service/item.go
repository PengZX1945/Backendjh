package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/repository"
)

type ItemService struct{}

func NewItemService() *ItemService {
	return &ItemService{}
}

type PublishParams struct {
	ItemName    string
	Category    string
	Location    string
	HappenTime  string
	Description string
	Images      []string
	GetContact  string
	GetLocation string
}

func (s *ItemService) Publish(userID uint64, typ string, p PublishParams) *errcode.Error {
	if typ != model.ItemTypeLost && typ != model.ItemTypeFound {
		return errcode.ParamError
	}
	if p.ItemName == "" || p.Category == "" {
		return errcode.ParamError
	}
	if typ == model.ItemTypeFound && p.GetContact == "" && p.GetLocation == "" {
		return errcode.ParamError
	}
	if len(p.Images) > 5 {
		return errcode.ParamError
	}
	item := &model.Item{
		Type:        typ,
		ItemName:    p.ItemName,
		Category:    p.Category,
		Location:    p.Location,
		HappenTime:  p.HappenTime,
		Description: p.Description,
		Status:      model.ItemStatusPending,
		PosterID:    userID,
		GetContact:  p.GetContact,
		GetLocation: p.GetLocation,
	}
	item.SetImages(p.Images)
	if err := repository.CreateItem(item); err != nil {
		return errcode.InternalError
	}
	return nil
}
