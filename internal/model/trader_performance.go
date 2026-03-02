package model

import "time"

// TraderPerformance 交易员绩效表（trader_performances）
type TraderPerformance struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID"`
	Address   string    `gorm:"type:varchar(42);not null;uniqueIndex:idx_perf_addr_window;comment:钱包地址"`
	Window    string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_perf_addr_window;comment:统计窗口（day/week/month/allTime）"`
	Pnl       string    `gorm:"type:numeric;comment:盈亏"`
	Roi       string    `gorm:"type:numeric;comment:收益率"`
	Vlm       string    `gorm:"type:numeric;comment:交易量"`
	CreatedAt time.Time `gorm:"comment:创建时间"`
	UpdatedAt time.Time `gorm:"comment:更新时间"`
}
