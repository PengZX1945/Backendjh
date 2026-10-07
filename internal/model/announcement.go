package model

import "time"

// Announcement 系统公告：仅系统管理员可创建 / 编辑 / 删除，所有用户可阅读
type Announcement struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedBy uint64    `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
