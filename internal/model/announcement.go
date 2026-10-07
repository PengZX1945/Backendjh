package model

// 公告状态：0 公开 / 1 已下线。
const (
	AnnouncementStatusPublished = 0 // 公开
	AnnouncementStatusOffline   = 1 // 已下线
)

// Announcement 是系统公告，仅系统管理员可维护。
type Announcement struct {
	AnnouncementID     uint   `gorm:"primaryKey;column:announcement_id" json:"announcement_id"`
	Title              string `gorm:"size:255;not null" json:"title"`
	Content            string `gorm:"type:text" json:"content"`
	AnnouncementStatus int    `gorm:"not null;default:0;index" json:"announcement_status"`
	CreatedTime        string `gorm:"size:20;not null;index" json:"created_time"`
}
