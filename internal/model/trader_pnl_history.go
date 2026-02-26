package model

import (
	"time"

	"gorm.io/datatypes"
)

// TraderPnlHistory 交易员盈亏历史
type TraderPnlHistory struct {
	ID        uint           `gorm:"primaryKey"`
	Address   string         `gorm:"type:varchar(42);not null;uniqueIndex:idx_pnl_addr_window"` // 钱包地址
	Window    string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_pnl_addr_window"` // day/week/month/allTime...
	History   datatypes.JSON `gorm:"type:jsonb"`                                                // [[timestamp, value], ...]
	CreatedAt time.Time
	UpdatedAt time.Time
}
