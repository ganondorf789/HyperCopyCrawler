package model

import (
	"time"

	"gorm.io/datatypes"
)

// TraderPnlHistory 交易员盈亏历史表（trader_pnl_histories）
type TraderPnlHistory struct {
	ID        uint           `gorm:"primaryKey;comment:主键ID"`
	Address   string         `gorm:"type:varchar(42);not null;uniqueIndex:idx_pnl_addr_window;comment:钱包地址"`
	Window    string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_pnl_addr_window;comment:统计窗口（day/week/month/allTime）"`
	History   datatypes.JSON `gorm:"type:jsonb;comment:盈亏历史数据（[[timestamp, value], ...]）"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
}
