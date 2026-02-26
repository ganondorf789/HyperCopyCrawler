package model

import "time"

// TraderPerformance 交易员绩效
type TraderPerformance struct {
	ID        uint           `gorm:"primaryKey"`
	Address   string         `gorm:"type:varchar(42);not null;uniqueIndex:idx_perf_addr_window"` // 钱包地址
	Window    string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_perf_addr_window"` // day/week/month/allTime
	Pnl       string         `gorm:"type:numeric"`                                               // 盈亏
	Roi       string         `gorm:"type:numeric"`                                               // 收益率
	Vlm       string         `gorm:"type:numeric"`                                               // 交易量
	CreatedAt time.Time
	UpdatedAt time.Time
}
