package model

// 物品大类：寻物启事 / 失物招领。
const (
	ItemTypeLost  = "lost"  // 寻物启事
	ItemTypeFound = "found" // 失物招领
)

// 物品状态机。create 落 0，审核通过落 1，驳回落 2，认领完成或线下确认后落 3。
const (
	ItemStatusPending   = 0 // 待审核
	ItemStatusPublished = 1 // 已发布
	ItemStatusRejected  = 2 // 已驳回
	ItemStatusClosed    = 3 // 已关闭（认领完成）
)

// IsValidItemType 校验大类取值，供发布/修改/列表查询共用。
func IsValidItemType(itemType string) bool {
	return itemType == ItemTypeLost || itemType == ItemTypeFound
}

// Item 是失物/招领帖子。
type Item struct {
	ItemID uint `gorm:"primaryKey;column:item_id" json:"item_id"`
	// Type 大类，取值见 ItemTypeLost / ItemTypeFound。
	Type string `gorm:"size:10;not null;index" json:"type"`
	// ItemName 物品名称。
	ItemName string `gorm:"size:128;not null" json:"item_name"`
	// Category 分类，取值见接口文档「通用约定」的九类。
	Category string `gorm:"size:32;not null" json:"category"`
	// Location 发现/遗失地点。
	Location string `gorm:"size:255" json:"location"`
	// HappenTime 发现/遗失时间。用字符串存 "YYYY-MM-DD HH:mm:ss"，字典序即时间序。
	HappenTime string `gorm:"size:20;index" json:"happen_time"`
	// PosterContact 发布时的联系方式快照。刻意冗余：发帖时留联系方式、之后改个人资料，
	// 历史帖子的联系入口不应随之变动。
	PosterContact string `gorm:"size:128" json:"poster_contact"`
	Description   string `gorm:"type:text" json:"description"`
	// Image 图片 URL 列表（0–5 张）。GORM 的 json serializer 负责 []string ↔ TEXT 的转换，
	// 避免为它单独写一套 Scanner/Valuer。
	Image        []string `gorm:"serializer:json" json:"image"`
	ItemStatus   int      `gorm:"not null;default:0;index" json:"item_status"`
	CreatedTime  string   `gorm:"size:20;not null;index" json:"created_time"`
	LastEditTime string   `gorm:"size:20;not null" json:"last_edit_time"`
	// RejectReason 审核驳回理由，仅在已驳回时非空。
	RejectReason string `gorm:"size:512" json:"reject_reason"`
	// PosterID 发布者用户 ID。
	PosterID    uint   `gorm:"not null;index" json:"poster_id"`
	GetLocation string `gorm:"size:255" json:"get_location"`
	GetContact  string `gorm:"size:128" json:"get_contact"`
}

// IsVisibleTo 判断某条帖子对指定访问者是否可见。
//
// 口径来自接口文档：「已发布/已关闭」对所有人可见；「待审核/已驳回」仅发布者本人
// 与后台管理员可见。viewer 传 nil 表示游客。
func (i *Item) IsVisibleTo(viewer *User) bool {
	if i.ItemStatus == ItemStatusPublished || i.ItemStatus == ItemStatusClosed {
		return true
	}
	if viewer == nil {
		return false
	}
	return viewer.UserID == i.PosterID || viewer.IsBackOffice()
}

// CanBeEdited 判断帖子当前状态是否允许修改。
// 已关闭的帖子不可再改（文档：仅待审核/已驳回/已发布可修改）。
func (i *Item) CanBeEdited() bool {
	return i.ItemStatus != ItemStatusClosed
}

// CanBeManagedBy 判断某用户是否有权修改或删除该帖子。
// 口径：发布者本人，或任一后台角色。
func (i *Item) CanBeManagedBy(user *User) bool {
	if user == nil {
		return false
	}
	return user.UserID == i.PosterID || user.IsBackOffice()
}
