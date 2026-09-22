package model

import "time"

const (
	RoleUser        = "user"
	RoleFinderAdmin = "finder_admin"
	RoleSysAdmin    = "sys_admin"
)

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:32;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:128;not null" json:"-"` // 永远不输出到 JSON
	Nickname     string    `gorm:"size:32;not null;default:''" json:"nickname"`
	Contact      string    `gorm:"size:64;not null;default:''" json:"contact"`
	Role         string    `gorm:"size:20;not null;default:'user'" json:"role"`
	Status       int8      `gorm:"not null;default:1" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"-"`
}
