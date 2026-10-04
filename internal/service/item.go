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

func (s *ItemService) MyItem(userID uint64, typ string, itemStatus *int8, page int, pageSize int) ([]model.Item, *errcode.Error) {
	if typ != "" && typ != model.ItemTypeLost && typ != model.ItemTypeFound {
		return nil, errcode.ParamError
	}
	if itemStatus != nil && (*itemStatus < 0 || *itemStatus > model.ItemStatusClaimed) {
		return nil, errcode.ParamError
	}
	itemFilter := repository.ItemListFilter{Type: typ, PosterID: &userID, Status: itemStatus, Page: page, PageSize: pageSize}
	itemList, err := repository.ListItems(itemFilter)
	if err != nil {
		return nil, errcode.InternalError
	}
	return itemList, nil
}

func (s *ItemService) Delete(userID uint64, role string, itemID uint64) *errcode.Error {
	if itemID == 0 {
		return errcode.ParamError
	}
	item, err := repository.FindItemByID(itemID)
	if err != nil {
		return errcode.InternalError
	}
	if item == nil {
		return errcode.NotFound
	}
	if item.PosterID != userID && role != model.RoleSysAdmin && role != model.RoleFinderAdmin {
		return errcode.Forbidden
	}
	err = repository.DeleteItem(itemID)
	if err != nil {
		return errcode.InternalError
	}
	return nil
}

type ListParams struct {
	Typ       string
	Category  string
	Location  string
	Keyword   string
	Page      int
	PageSize  int
	StartTime string
	EndTime   string
}

func (s *ItemService) ListItems(l ListParams) ([]model.Item, *errcode.Error) {
	if l.Typ != model.ItemTypeLost && l.Typ != model.ItemTypeFound {
		return nil, errcode.ParamError
	}
	status := model.ItemStatusPublished
	itemFilter := repository.ItemListFilter{Type: l.Typ, Status: &status, Page: l.Page, PageSize: l.PageSize, StartTime: l.StartTime, EndTime: l.EndTime, Category: l.Category, Location: l.Location, Keyword: l.Keyword}
	itemList, err := repository.ListItems(itemFilter)
	if err != nil {
		return nil, errcode.InternalError
	}
	return itemList, nil
}

func (s *ItemService) Detail(userID uint64, role string, itemID uint64) (*model.Item, *errcode.Error) {
	item, err := repository.FindItemByID(itemID)
	if err != nil {
		return nil, errcode.InternalError
	}
	if item == nil {
		return nil, errcode.NotFound
	}
	if item.Status == model.ItemStatusPublished || item.Status == model.ItemStatusClaimed {
		return item, nil
	}
	if userID != 0 && item.PosterID == userID || role == model.RoleSysAdmin || role == model.RoleFinderAdmin {
		return item, nil
	}
	return nil, errcode.NotFound
}
