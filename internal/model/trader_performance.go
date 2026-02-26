package model

import (
	"time"

	"gorm.io/datatypes"
)

// TraderPerformance 交易员绩效
type TraderPerformance struct {
	ID                 uint           `gorm:"primaryKey"`
	Address            string         `gorm:"type:varchar(42);not null;uniqueIndex"` // 钱包地址
	WindowPerformances datatypes.JSON `gorm:"type:jsonb"`                            // 各时间窗口绩效
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
