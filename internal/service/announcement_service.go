package service

import (
	"strings"

	"lostfound/internal/dto"
	"lostfound/internal/model"
	"lostfound/internal/repository"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
	"lostfound/pkg/timeutil"
)

// AnnouncementService 处理公告的公开查询与管理端维护。
type AnnouncementService struct {
	announcements *repository.AnnouncementRepository
}

// NewAnnouncementService 构造公告服务。
func NewAnnouncementService(announcements *repository.AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{announcements: announcements}
}

// ListPublic 公开公告列表：只返回「公开」状态。
func (s *AnnouncementService) ListPublic(query dto.AnnouncementQuery) (dto.AnnouncementListResponse, error) {
	published := model.AnnouncementStatusPublished
	query.Status = &published

	announcements, err := s.announcements.List(query.Status, query.Offset(), query.PageSize)
	if err != nil {
		return dto.AnnouncementListResponse{}, err
	}
	return dto.NewAnnouncementListResponse(announcements), nil
}

// Detail 公告详情。
func (s *AnnouncementService) Detail(announcementID uint) (dto.AnnouncementResponse, error) {
	announcement, err := s.announcements.FindByID(announcementID)
	if err != nil {
		return dto.AnnouncementResponse{}, translate(err)
	}
	return dto.NewAnnouncementResponse(announcement), nil
}

// ListAll 全部公告（含已下线），仅供系统管理员。
func (s *AnnouncementService) ListAll(query dto.AnnouncementQuery) (dto.AnnouncementListResponse, error) {
	announcements, err := s.announcements.List(query.Status, query.Offset(), query.PageSize)
	if err != nil {
		return dto.AnnouncementListResponse{}, err
	}
	return dto.NewAnnouncementListResponse(announcements), nil
}

// Create 发布公告，落「公开」。
func (s *AnnouncementService) Create(req dto.AnnouncementCreateRequest) error {
	if err := requireText(req.Title, req.Content); err != nil {
		return err
	}
	return s.announcements.Create(&model.Announcement{
		Title:              req.Title,
		Content:            req.Content,
		AnnouncementStatus: model.AnnouncementStatusPublished,
		CreatedTime:        timeutil.Now(),
	})
}

// Update 修改公告。
//
// 只覆盖请求里显式提交的字段（用指针区分「没提交」与「提交了空串」）；上下线状态
// 也借这个接口承载 —— 文档给出了「已下线」状态却没有独立的上下线接口。
func (s *AnnouncementService) Update(announcementID uint, req dto.AnnouncementUpdateRequest) error {
	announcement, err := s.announcements.FindByID(announcementID)
	if err != nil {
		return translate(err)
	}

	if req.Title != nil {
		announcement.Title = *req.Title
	}
	if req.Content != nil {
		announcement.Content = *req.Content
	}
	if req.AnnouncementStatus != nil {
		// 只认 1（已下线），其余一律按 0（公开）处理，与数据模型的两态定义一致。
		announcement.AnnouncementStatus = model.AnnouncementStatusPublished
		if *req.AnnouncementStatus == model.AnnouncementStatusOffline {
			announcement.AnnouncementStatus = model.AnnouncementStatusOffline
		}
	}

	// 标题与正文不允许被改成空：置空等于把公告变成一条无法阅读的记录。
	// 这里校验的是「改完之后」的值，因此只传 announcement_status 的上下线请求不受影响。
	if strings.TrimSpace(announcement.Title) == "" || strings.TrimSpace(announcement.Content) == "" {
		return apperr.New(errcode.BadRequest)
	}

	return s.announcements.Update(announcement)
}

// Delete 删除公告。
func (s *AnnouncementService) Delete(announcementID uint) error {
	if _, err := s.announcements.FindByID(announcementID); err != nil {
		return translate(err)
	}
	return s.announcements.Delete(announcementID)
}
