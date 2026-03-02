package model

import (
	"time"

	"gorm.io/datatypes"
)

// TraderAccountValue 交易员账户价值历史表（trader_account_values）
type TraderAccountValue struct {
	ID        uint           `gorm:"primaryKey;comment:主键ID"`
	Address   string         `gorm:"type:varchar(42);not null;uniqueIndex:idx_acct_addr_window;comment:钱包地址"`
	Window    string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_acct_addr_window;comment:统计窗口（day/week/month/allTime）"`
	History   datatypes.JSON `gorm:"type:jsonb;comment:账户价值历史数据（[[timestamp, value], ...]）"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
}
