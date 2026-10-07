// Package dto 定义对外传输对象：请求体、响应体与查询条件。
//
// 存在的意义是把「接口字段名」与「数据库字段名」分开。文档要求接口用下划线
// （item_name、happen_time），而 Go 结构体内部用大驼峰；两者的映射只在本文件与
// model 的 json tag 上各声明一次，service / repository 不必关心命名的差异。
package dto

import "lostfound/internal/model"

/* ────────────────────────── 用户与鉴权 ────────────────────────── */

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Contact  string `json:"contact"`
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应体：凭证固定在 data.token。
//
// 前端原本按候选字段名探测 token（token/access_token/jwt…），后端定稿后固定为
// token，前端只需保留候选数组里的第一项即可。
type LoginResponse struct {
	Token string `json:"token"`
}

// ProfileResponse 是当前登录用户的档案。
//
// 字段名取文档 GET /auth/profile 示例里的 id：数据模型的通用「用户信息」用 user_id，
// 而档案接口示例用 id。前端两者都兼容，此处按示例对齐。
type ProfileResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	Contact  string `json:"contact"`
}

// UpdateProfileRequest 修改昵称与联系方式（用户名、角色不可改）。
type UpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Contact  string `json:"contact"`
}

// ChangePasswordRequest 修改密码。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ManagedUserResponse 是管理端用户列表的一行，按数据模型使用 user_id。
type ManagedUserResponse struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	Contact  string `json:"contact"`
}

// UpdateUserRoleRequest 调整用户角色，只允许 finder_admin 与 user。
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// UserListResponse 用户列表信封。
type UserListResponse struct {
	Users []ManagedUserResponse `json:"users"`
}

/* ────────────────────────── 物品 ────────────────────────── */

// ItemRequest 是发布与修改物品共用的请求体。
//
// 修改接口是全段更新：前端会把所有字段一并提交，未填的可选字段显式给空串，
// 因此这里不对可选字段做「缺省保留旧值」处理，一律按提交值覆盖。
type ItemRequest struct {
	Type        string   `json:"type"`
	ItemName    string   `json:"item_name"`
	Category    string   `json:"category"`
	Location    string   `json:"location"`
	HappenTime  string   `json:"happen_time"`
	Description string   `json:"description"`
	Image       []string `json:"image"`
	GetLocation string   `json:"get_location"`
	GetContact  string   `json:"get_contact"`
}

// ItemResponse 是物品的对外表示，字段与数据模型「物品信息」逐条对应。
type ItemResponse struct {
	ItemID        uint     `json:"item_id"`
	Type          string   `json:"type"`
	ItemName      string   `json:"item_name"`
	Category      string   `json:"category"`
	Location      string   `json:"location"`
	HappenTime    string   `json:"happen_time"`
	PosterContact string   `json:"poster_contact"`
	Description   string   `json:"description"`
	Image         []string `json:"image"`
	ItemStatus    int      `json:"item_status"`
	CreatedTime   string   `json:"created_time"`
	LastEditTime  string   `json:"last_edit_time"`
	RejectReason  string   `json:"reject_reason"`
	PosterID      uint     `json:"poster_id"`
	GetLocation   string   `json:"get_location"`
	GetContact    string   `json:"get_contact"`
}

// ItemListResponse 物品列表信封。
type ItemListResponse struct {
	Items []ItemResponse `json:"items"`
}

// NewItemResponse 把实体映射为响应体。
func NewItemResponse(item *model.Item) ItemResponse {
	images := item.Image
	if images == nil {
		// 空图集显式序列化成 []，而不是 null —— 前端的图片渲染直接对它做 map。
		images = []string{}
	}
	return ItemResponse{
		ItemID:        item.ItemID,
		Type:          item.Type,
		ItemName:      item.ItemName,
		Category:      item.Category,
		Location:      item.Location,
		HappenTime:    item.HappenTime,
		PosterContact: item.PosterContact,
		Description:   item.Description,
		Image:         images,
		ItemStatus:    item.ItemStatus,
		CreatedTime:   item.CreatedTime,
		LastEditTime:  item.LastEditTime,
		RejectReason:  item.RejectReason,
		PosterID:      item.PosterID,
		GetLocation:   item.GetLocation,
		GetContact:    item.GetContact,
	}
}

// NewItemListResponse 批量映射。
// 始终初始化成非 nil 切片：列表为空时序列化成 []，避免前端拿到 null 后要多写一层判空。
func NewItemListResponse(items []model.Item) ItemListResponse {
	records := make([]ItemResponse, 0, len(items))
	for index := range items {
		records = append(records, NewItemResponse(&items[index]))
	}
	return ItemListResponse{Items: records}
}

/* ────────────────────────── 认领申请 ────────────────────────── */

// ClaimSubmitRequest 提交认领申请的请求体（物品 ID 走 query）。
type ClaimSubmitRequest struct {
	Reason string `json:"reason"`
}

// ClaimUpdateRequest 修改认领申请的请求体。
type ClaimUpdateRequest struct {
	Reason           string `json:"reason"`
	ApplicantContact string `json:"applicant_contact"`
}

// ClaimResponse 是认领申请列表的一行。
type ClaimResponse struct {
	ClaimID          uint   `json:"claim_id"`
	ItemID           uint   `json:"item_id"`
	Reason           string `json:"reason"`
	ApplicantContact string `json:"applicant_contact"`
	CreatedTime      string `json:"created_time"`
	LastEditTime     string `json:"last_edit_time"`
	ClaimStatus      int    `json:"claim_status"`
	ApplicantID      uint   `json:"applicant_id"`
}

// ClaimDetailResponse 是认领申请详情，内联了对应物品。
type ClaimDetailResponse struct {
	ClaimID      uint          `json:"claim_id"`
	Item         *ItemResponse `json:"item"`
	Reason       string        `json:"reason"`
	Contact      string        `json:"contact"`
	CreatedTime  string        `json:"created_time"`
	LastEditTime string        `json:"last_edit_time"`
	ClaimStatus  int           `json:"claim_status"`
	UserID       uint          `json:"user_id"`
}

// ClaimListResponse 认领申请列表信封。
type ClaimListResponse struct {
	Claims []ClaimResponse `json:"claims"`
}

// NewClaimResponse 把实体映射为列表行。
func NewClaimResponse(claim *model.Claim) ClaimResponse {
	return ClaimResponse{
		ClaimID:          claim.ClaimID,
		ItemID:           claim.ItemID,
		Reason:           claim.Reason,
		ApplicantContact: claim.ApplicantContact,
		CreatedTime:      claim.CreatedTime,
		LastEditTime:     claim.LastEditTime,
		ClaimStatus:      claim.ClaimStatus,
		ApplicantID:      claim.ApplicantID,
	}
}

// NewClaimListResponse 批量映射。
func NewClaimListResponse(claims []model.Claim) ClaimListResponse {
	records := make([]ClaimResponse, 0, len(claims))
	for index := range claims {
		records = append(records, NewClaimResponse(&claims[index]))
	}
	return ClaimListResponse{Claims: records}
}

/* ────────────────────────── 公告 ────────────────────────── */

// AnnouncementCreateRequest 发布公告的请求体，标题与正文必填。
type AnnouncementCreateRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// AnnouncementUpdateRequest 修改公告的请求体，字段全部可选（只改提交了的字段）。
//
// 用指针而非字符串来区分「没提交」与「提交了空串」：前者应保留原值，
// 后者是明确的清空意图。上下线状态也借这个接口承载（文档没有独立的上下线接口）。
type AnnouncementUpdateRequest struct {
	Title              *string `json:"title"`
	Content            *string `json:"content"`
	AnnouncementStatus *int    `json:"announcement_status"`
}

// AnnouncementResponse 是公告的对外表示。
type AnnouncementResponse struct {
	AnnouncementID     uint   `json:"announcement_id"`
	Title              string `json:"title"`
	Content            string `json:"content"`
	AnnouncementStatus int    `json:"announcement_status"`
	CreatedTime        string `json:"created_time"`
}

// AnnouncementListResponse 公告列表信封。
type AnnouncementListResponse struct {
	Announcements []AnnouncementResponse `json:"announcements"`
}

// NewAnnouncementResponse 把实体映射为响应体。
func NewAnnouncementResponse(announcement *model.Announcement) AnnouncementResponse {
	return AnnouncementResponse{
		AnnouncementID:     announcement.AnnouncementID,
		Title:              announcement.Title,
		Content:            announcement.Content,
		AnnouncementStatus: announcement.AnnouncementStatus,
		CreatedTime:        announcement.CreatedTime,
	}
}

// NewAnnouncementListResponse 批量映射。
func NewAnnouncementListResponse(announcements []model.Announcement) AnnouncementListResponse {
	records := make([]AnnouncementResponse, 0, len(announcements))
	for index := range announcements {
		records = append(records, NewAnnouncementResponse(&announcements[index]))
	}
	return AnnouncementListResponse{Announcements: records}
}

/* ────────────────────────── 管理端专用 ────────────────────────── */

// ItemReviewRequest 是审核请求体。仅驳回时有意义 —— 通过时前端不传请求体。
type ItemReviewRequest struct {
	RejectReason string `json:"reject_reason"`
}

// ReviewItemResponse 审核结果，回传落库后的驳回理由。
type ReviewItemResponse struct {
	RejectReason string `json:"reject_reason"`
}

// SetItemStatusRequest 管理端调整物品状态。文档限定只能改为 3（已关闭）。
type SetItemStatusRequest struct {
	Status int `json:"status"`
}

// UploadResponse 上传成功后返回可访问的图片地址。
type UploadResponse struct {
	URL string `json:"url"`
}

/* ────────────────────────── 查询条件 ────────────────────────── */

// PageQuery 是分页的公共入参。
type PageQuery struct {
	Page     int
	PageSize int
}

// Offset 返回 SQL 偏移量。
func (p PageQuery) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// ItemQuery 是物品列表的查询条件。
// 同一套条件同时服务于公开信息流、我的发布与管理端总览，只是各接口填其中一部分。
type ItemQuery struct {
	PageQuery
	Type      string
	Status    *int
	Category  string
	Keyword   string
	Location  string
	StartTime string
	EndTime   string
	PosterID  *uint
}

// ClaimQuery 是认领申请的查询条件。
type ClaimQuery struct {
	PageQuery
	Status      *int
	ItemID      *uint
	ApplicantID *uint
}

// AnnouncementQuery 是公告的查询条件。
type AnnouncementQuery struct {
	PageQuery
	Status *int
}

// UserQuery 是用户列表的查询条件。
type UserQuery struct {
	PageQuery
	Role    string
	Keyword string
}
