package model

import (
	"encoding/json"
	"time"
)

type Item struct {
	ID           uint64    `gorm:"primaryKey" json:"item_id"`
	Type         string    `gorm:"size:8;not null;index" json:"type"`
	ItemName     string    `gorm:"size:64;not null" json:"item_name"`
	Category     string    `gorm:"size:32;not null" json:"category"`
	Location     string    `gorm:"size:128" json:"location"`
	HappenTime   string    `gorm:"size:32" json:"happen_time"`
	Description  string    `gorm:"size:1024" json:"description"`
	Images       string    `gorm:"size:2048" json:"-"`
	Status       int8      `gorm:"not null;default:0;index" json:"item_status"`
	RejectReason string    `gorm:"size:255" json:"reject_reason"`
	PosterID     uint64    `gorm:"not null;index" json:"poster_id"`
	CreatedAt    time.Time `json:"created_time"`
	UpdatedAt    time.Time `json:"last_edit_time"`
	GetContact   string    `gorm:"size:64" json:"get_contact"`
	GetLocation  string    `gorm:"size:128" json:"get_location"`
}

const (
	ItemStatusPending   = 0 // 待审核
	ItemStatusPublished = 1 // 已发布
	ItemStatusRejected  = 2 // 已驳回
	ItemStatusClaimed   = 3 // 已认领
)

const (
	ItemTypeLost  = "lost"
	ItemTypeFound = "found"
)

func (it *Item) SetImages(urls []string) {
	b, _ := json.Marshal(urls)
	it.Images = string(b)
}

func (it *Item) GetImages() []string {
	var urls []string
	json.Unmarshal([]byte(it.Images), &urls)
	return urls
}
