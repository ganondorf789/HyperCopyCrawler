package model

import "time"

// CompletedTrade 已完成的交易（从 fills 聚合而来）
type CompletedTrade struct {
	ID         uint    `gorm:"primaryKey"`
	Address    string  `gorm:"type:varchar(42);not null;index:idx_ct_addr"`  // 钱包地址
	Coin       string  `gorm:"type:varchar(20);not null"`                    // 币种
	MarginMode string  `gorm:"type:varchar(20);not null"`                    // isolated / cross
	Direction  string  `gorm:"type:varchar(10);not null"`                    // long / short
	Size       float64 `gorm:"type:numeric;not null"`                        // 最大持仓量
	EntryPrice float64 `gorm:"type:numeric;not null"`                        // 加权平均入场价
	ClosePrice float64 `gorm:"type:numeric;not null"`                        // 加权平均平仓价
	StartTime  int64   `gorm:"not null;index:idx_ct_addr"`                   // 开仓时间（毫秒）
	EndTime    int64   `gorm:"not null"`                                     // 平仓时间（毫秒）
	TotalFee   float64 `gorm:"type:numeric;not null;default:0"`              // 总手续费
	Pnl        float64 `gorm:"type:numeric;not null;default:0"`              // 已实现盈亏（closedPnl 之和）
	FillCount  int     `gorm:"not null;default:0"`                           // 成交笔数
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
