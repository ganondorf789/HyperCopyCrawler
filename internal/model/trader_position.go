package model

import "time"

// TraderPosition 交易员持仓表（trader_positions）
type TraderPosition struct {
	ID                    uint   `gorm:"primaryKey;comment:主键ID"`
	Address               string `gorm:"type:varchar(42);not null;uniqueIndex:idx_pos_addr_coin;comment:钱包地址"`
	Coin                  string `gorm:"type:varchar(20);not null;uniqueIndex:idx_pos_addr_coin;comment:币种"`
	Szi                   string `gorm:"type:numeric;not null;comment:仓位大小（正值为多头，负值为空头）"`
	LeverageType          string `gorm:"type:varchar(20);not null;comment:杠杆类型（cross/isolated）"`
	Leverage              int    `gorm:"not null;comment:杠杆倍数"`
	EntryPx               string `gorm:"type:numeric;not null;comment:入场价"`
	PositionValue         string `gorm:"type:numeric;not null;comment:持仓价值"`
	UnrealizedPnl         string `gorm:"type:numeric;comment:未实现盈亏"`
	ReturnOnEquity        string `gorm:"type:numeric;comment:权益回报率"`
	LiquidationPx         string `gorm:"type:numeric;comment:清算价"`
	MarginUsed            string `gorm:"type:numeric;comment:已用保证金"`
	MaxLeverage           int    `gorm:"not null;comment:最大允许杠杆"`
	CumFundingAllTime     string `gorm:"type:numeric;comment:累计资金费（全部时间）"`
	CumFundingSinceOpen   string `gorm:"type:numeric;comment:累计资金费（开仓以来）"`
	CumFundingSinceChange string `gorm:"type:numeric;comment:累计资金费（最近变更以来）"`
	CreatedAt             time.Time `gorm:"comment:创建时间"`
	UpdatedAt             time.Time `gorm:"comment:更新时间"`
}
