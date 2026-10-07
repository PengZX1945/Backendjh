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

// maxItemImages 是单帖图片数上限，来自接口文档（0–5 张）。
const maxItemImages = 5

// 列表排序口径。放在一处是为了说明「为什么不同接口排的字段不一样」：
// 公开信息流按「发现/遗失时间」排（用户关心东西什么时候丢的），
// 其余按「创建时间」排（管理台与个人列表关心提交先后）。
const (
	orderByHappenTime  = "happen_time desc"
	orderByCreatedTime = "created_time desc"
)

// ItemService 处理物品的发布、修改、查询与审核。
type ItemService struct {
	items *repository.ItemRepository
}

// NewItemService 构造物品服务。
func NewItemService(items *repository.ItemRepository) *ItemService {
	return &ItemService{items: items}
}

// itemFilter 把查询条件翻译成仓储层 filter。
func itemFilter(query dto.ItemQuery, orderBy string) repository.ItemFilter {
	return repository.ItemFilter{
		Type:      query.Type,
		Status:    query.Status,
		Category:  query.Category,
		Keyword:   query.Keyword,
		Location:  query.Location,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		PosterID:  query.PosterID,
		OrderBy:   orderBy,
		Offset:    query.Offset(),
		Limit:     query.PageSize,
	}
}

// ListPublished 公开信息流：仅已发布，按发现/遗失时间倒序。
func (s *ItemService) ListPublished(itemType string, query dto.ItemQuery) (dto.ItemListResponse, error) {
	if !model.IsValidItemType(itemType) {
		return dto.ItemListResponse{}, apperr.New(errcode.BadRequest)
	}

	published := model.ItemStatusPublished
	query.Type = itemType
	query.Status = &published

	items, err := s.items.List(itemFilter(query, orderByHappenTime))
	if err != nil {
		return dto.ItemListResponse{}, err
	}
	return dto.NewItemListResponse(items), nil
}

// Detail 物品详情。
//
// 待审核/已驳回的帖子只对发布者本人与后台角色可见，其余访问者一律按「资源不存在」
// 处理 —— 用 404 而不是 403，避免通过状态码探测出「这个 ID 存在但没通过审核」。
func (s *ItemService) Detail(viewer *model.User, itemID uint) (dto.ItemResponse, error) {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return dto.ItemResponse{}, translate(err)
	}
	if !item.IsVisibleTo(viewer) {
		return dto.ItemResponse{}, apperr.New(errcode.NotFound)
	}
	return dto.NewItemResponse(item), nil
}

// Create 发布物品，落「待审核」，审核通过后才在公开信息流出现。
func (s *ItemService) Create(user *model.User, itemType string, req dto.ItemRequest) error {
	if !model.IsValidItemType(itemType) {
		return apperr.New(errcode.BadRequest)
	}
	if err := validateItemRequest(req); err != nil {
		return err
	}

	now := timeutil.Now()
	happenTime := strings.TrimSpace(req.HappenTime)
	if happenTime == "" {
		// 未填发现/遗失时间时以发布时刻兜底：信息流正是按该字段倒序，
		// 留空会让这条帖子在排序中失去位置。
		happenTime = now
	}

	return s.items.Create(&model.Item{
		Type:          itemType,
		ItemName:      req.ItemName,
		Category:      req.Category,
		Location:      req.Location,
		HappenTime:    happenTime,
		PosterContact: user.Contact,
		Description:   req.Description,
		Image:         normalizeImages(req.Image),
		ItemStatus:    model.ItemStatusPending,
		CreatedTime:   now,
		LastEditTime:  now,
		PosterID:      user.UserID,
		GetLocation:   req.GetLocation,
		GetContact:    req.GetContact,
	})
}

// Update 修改物品。接口约定为全段更新，改完状态重置为「待审核」。
func (s *ItemService) Update(user *model.User, itemID uint, req dto.ItemRequest) error {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return translate(err)
	}
	if !item.CanBeManagedBy(user) {
		return apperr.New(errcode.Forbidden)
	}
	if !item.CanBeEdited() {
		return apperr.New(errcode.InvalidState)
	}
	if err := validateItemRequest(req); err != nil {
		return err
	}

	// 大类允许随修改一并调整，但必须仍是合法取值；未提交则保留原大类。
	if trimmedType := strings.TrimSpace(req.Type); trimmedType != "" {
		if !model.IsValidItemType(trimmedType) {
			return apperr.New(errcode.BadRequest)
		}
		item.Type = trimmedType
	}

	item.ItemName = req.ItemName
	item.Category = req.Category
	item.Location = req.Location
	item.HappenTime = req.HappenTime
	item.Description = req.Description
	item.Image = normalizeImages(req.Image)
	item.GetLocation = req.GetLocation
	item.GetContact = req.GetContact
	// 已发布的内容被改动后必须重新过审 —— 否则「先发正常内容再改成违规内容」
	// 就成了绕过审核的通路。
	item.ItemStatus = model.ItemStatusPending
	item.RejectReason = ""
	item.LastEditTime = timeutil.Now()

	return s.items.Update(item)
}

// Delete 删除物品。本人或任一后台角色可删。
func (s *ItemService) Delete(user *model.User, itemID uint) error {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return translate(err)
	}
	if !item.CanBeManagedBy(user) {
		return apperr.New(errcode.Forbidden)
	}
	return s.items.Delete(itemID)
}

// ListMine 我的发布：含全部状态，并带上驳回理由供用户修改后重新提交。
func (s *ItemService) ListMine(userID uint, query dto.ItemQuery) (dto.ItemListResponse, error) {
	query.PosterID = &userID

	items, err := s.items.List(itemFilter(query, orderByCreatedTime))
	if err != nil {
		return dto.ItemListResponse{}, err
	}
	return dto.NewItemListResponse(items), nil
}

// CloseClaimChannel 关闭认领通道：本人或任一后台角色。
func (s *ItemService) CloseClaimChannel(user *model.User, itemID uint) error {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return translate(err)
	}
	if !item.CanBeManagedBy(user) {
		return apperr.New(errcode.Forbidden)
	}
	return s.items.UpdateStatus(itemID, model.ItemStatusClosed, timeutil.Now())
}

// ListPending 待审核列表（审核台）。
func (s *ItemService) ListPending(itemType string, query dto.ItemQuery) (dto.ItemListResponse, error) {
	if itemType != "" && !model.IsValidItemType(itemType) {
		return dto.ItemListResponse{}, apperr.New(errcode.BadRequest)
	}

	pending := model.ItemStatusPending
	query.Type = itemType
	query.Status = &pending

	items, err := s.items.List(itemFilter(query, orderByCreatedTime))
	if err != nil {
		return dto.ItemListResponse{}, err
	}
	return dto.NewItemListResponse(items), nil
}

// ListAll 全校总览（系统管理员），可按类型/状态/分类/关键词过滤。
func (s *ItemService) ListAll(query dto.ItemQuery) (dto.ItemListResponse, error) {
	if query.Type != "" && !model.IsValidItemType(query.Type) {
		return dto.ItemListResponse{}, apperr.New(errcode.BadRequest)
	}

	items, err := s.items.List(itemFilter(query, orderByCreatedTime))
	if err != nil {
		return dto.ItemListResponse{}, err
	}
	return dto.NewItemListResponse(items), nil
}

// Review 审核：通过落「已发布」，驳回落「已驳回」并记录理由。
// 只允许审核「待审核」的帖子，其余判状态不允许(7)，避免重复审核覆盖既有结论。
func (s *ItemService) Review(itemID uint, option, rejectReason string) (dto.ReviewItemResponse, error) {
	item, err := s.items.FindByID(itemID)
	if err != nil {
		return dto.ReviewItemResponse{}, translate(err)
	}
	if item.ItemStatus != model.ItemStatusPending {
		return dto.ReviewItemResponse{}, apperr.New(errcode.InvalidState)
	}

	status := model.ItemStatusPublished
	reason := ""
	switch option {
	case "approve":
		// 通过时清空历史驳回理由，避免重新过审的帖子还挂着上一轮的原因。
	case "reject":
		status = model.ItemStatusRejected
		reason = strings.TrimSpace(rejectReason)
	default:
		return dto.ReviewItemResponse{}, apperr.New(errcode.BadRequest)
	}

	item.ItemStatus = status
	item.RejectReason = reason
	item.LastEditTime = timeutil.Now()
	if err := s.items.Update(item); err != nil {
		return dto.ReviewItemResponse{}, err
	}
	return dto.ReviewItemResponse{RejectReason: reason}, nil
}

// SetStatusClosed 管理端调整物品状态，仅支持置为「已关闭」（线下确认后关闭）。
func (s *ItemService) SetStatusClosed(itemID uint, status int) error {
	// 先判存在再判取值：物品不存在时应回 4 而不是 1，否则调用方无从区分。
	if _, err := s.items.FindByID(itemID); err != nil {
		return translate(err)
	}
	if status != model.ItemStatusClosed {
		return apperr.New(errcode.BadRequest)
	}
	return s.items.UpdateStatus(itemID, model.ItemStatusClosed, timeutil.Now())
}

// validateItemRequest 校验发布/修改的必填项与图片数量。
func validateItemRequest(req dto.ItemRequest) error {
	if err := requireText(req.ItemName, req.Category, req.GetLocation, req.GetContact); err != nil {
		return err
	}
	if len(req.Image) > maxItemImages {
		return apperr.New(errcode.BadRequest)
	}
	return nil
}

// normalizeImages 归一化图片列表：过滤空串与纯空白，并把 nil 收敛为空切片。
// 空切片而非 nil 落库，保证接口输出恒为 []，前端可无条件遍历。
func normalizeImages(images []string) []string {
	normalized := make([]string, 0, len(images))
	for _, image := range images {
		if trimmed := strings.TrimSpace(image); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}
