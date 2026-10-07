package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/repository"
)

type AnnouncementService struct{}

func NewAnnouncementService() *AnnouncementService { return &AnnouncementService{} }

// List 公开：所有人可查看最新公告
func (s *AnnouncementService) List(limit int) ([]model.Announcement, *errcode.Error) {
	list, err := repository.ListAnnouncements(limit)
	if err != nil {
		return nil, errcode.InternalError
	}
	return list, nil
}

// Create 仅系统管理员
func (s *AnnouncementService) Create(operatorRole, title, content string, operatorID uint64) (*model.Announcement, *errcode.Error) {
	if operatorRole != model.RoleSysAdmin {
		return nil, errcode.Forbidden
	}
	if title == "" || content == "" {
		return nil, errcode.ParamError
	}
	a := &model.Announcement{
		Title:     title,
		Content:   content,
		CreatedBy: operatorID,
	}
	if err := repository.CreateAnnouncement(a); err != nil {
		return nil, errcode.InternalError
	}
	return a, nil
}

// Update 仅系统管理员，可改标题与内容
func (s *AnnouncementService) Update(operatorRole string, id uint64, title, content string) *errcode.Error {
	if operatorRole != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	if title == "" || content == "" {
		return errcode.ParamError
	}
	a, err := repository.FindAnnouncementByID(id)
	if err != nil {
		return errcode.InternalError
	}
	if a == nil {
		return errcode.NotFound
	}
	a.Title = title
	a.Content = content
	if err := repository.UpdateAnnouncement(a); err != nil {
		return errcode.InternalError
	}
	return nil
}

// Delete 仅系统管理员
func (s *AnnouncementService) Delete(operatorRole string, id uint64) *errcode.Error {
	if operatorRole != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	a, err := repository.FindAnnouncementByID(id)
	if err != nil {
		return errcode.InternalError
	}
	if a == nil {
		return errcode.NotFound
	}
	if err := repository.DeleteAnnouncement(id); err != nil {
		return errcode.InternalError
	}
	return nil
}
