package model

import (
	"time"

	"gorm.io/datatypes"
)

// TraderAccountValue 交易员账户价值历史
type TraderAccountValue struct {
	ID        uint           `gorm:"primaryKey"`
	Address   string         `gorm:"type:varchar(42);not null;uniqueIndex:idx_acct_addr_window"` // 钱包地址
	Window    string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_acct_addr_window"` // day/week/month/allTime...
	History   datatypes.JSON `gorm:"type:jsonb"`                                                 // [[timestamp, value], ...]
	CreatedAt time.Time
	UpdatedAt time.Time
}
