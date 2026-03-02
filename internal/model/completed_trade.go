package model

import "time"

// CompletedTrade 已完成交易表，由 fills 聚合而来（completed_trades）
type CompletedTrade struct {
	ID         uint    `gorm:"primaryKey;comment:主键ID"`
	Address    string  `gorm:"type:varchar(42);not null;index:idx_ct_addr;comment:钱包地址"`
	Coin       string  `gorm:"type:varchar(20);not null;comment:币种"`
	MarginMode string  `gorm:"type:varchar(20);not null;comment:保证金模式（isolated/cross）"`
	Direction  string  `gorm:"type:varchar(10);not null;comment:方向（long/short）"`
	Size       float64 `gorm:"type:numeric;not null;comment:最大持仓量"`
	EntryPrice float64 `gorm:"type:numeric;not null;comment:加权平均入场价"`
	ClosePrice float64 `gorm:"type:numeric;not null;comment:加权平均平仓价"`
	StartTime  int64   `gorm:"not null;index:idx_ct_addr;comment:开仓时间（毫秒时间戳）"`
	EndTime    int64   `gorm:"not null;comment:平仓时间（毫秒时间戳）"`
	TotalFee   float64 `gorm:"type:numeric;not null;default:0;comment:总手续费"`
	Pnl        float64 `gorm:"type:numeric;not null;default:0;comment:已实现盈亏（closedPnl 之和）"`
	FillCount  int     `gorm:"not null;default:0;comment:成交笔数"`
	CreatedAt  time.Time `gorm:"comment:创建时间"`
	UpdatedAt  time.Time `gorm:"comment:更新时间"`
}
