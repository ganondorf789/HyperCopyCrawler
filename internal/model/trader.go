package model

import (
	"time"

	"github.com/lib/pq"
)

// Trader 交易员
type Trader struct {
	ID             uint           `gorm:"primaryKey"`
	TwitterName    string         `gorm:"type:varchar(255);not null"`                    // 推特显示名
	Username       string         `gorm:"type:varchar(255);not null;uniqueIndex"`        // 推特用户名
	Address        string         `gorm:"type:varchar(42);not null;uniqueIndex"`         // 钱包地址
	ProfilePicture string         `gorm:"type:text"`                                     // 头像链接
	Labels         pq.StringArray `gorm:"type:text[];default:'{}'"` // 标签列表
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
