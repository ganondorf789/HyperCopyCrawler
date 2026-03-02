package model

import "time"

// TraderPosition 交易员持仓
type TraderPosition struct {
	ID                    uint   `gorm:"primaryKey"`
	Address               string `gorm:"type:varchar(42);not null;uniqueIndex:idx_pos_addr_coin"` // 钱包地址
	Coin                  string `gorm:"type:varchar(20);not null;uniqueIndex:idx_pos_addr_coin"` // 币种
	Szi                   string `gorm:"type:numeric;not null"`                                   // 仓位大小（正多负空）
	LeverageType          string `gorm:"type:varchar(20);not null"`                               // 杠杆类型 cross/isolated
	Leverage              int    `gorm:"not null"`                                                // 杠杆倍数
	EntryPx               string `gorm:"type:numeric;not null"`                                   // 入场价
	PositionValue         string `gorm:"type:numeric;not null"`                                   // 持仓价值
	UnrealizedPnl         string `gorm:"type:numeric"`                                            // 未实现盈亏
	ReturnOnEquity        string `gorm:"type:numeric"`                                            // 权益回报率
	LiquidationPx         string `gorm:"type:numeric"`                                            // 清算价
	MarginUsed            string `gorm:"type:numeric"`                                            // 已用保证金
	MaxLeverage           int    `gorm:"not null"`                                                // 最大杠杆
	CumFundingAllTime     string `gorm:"type:numeric"`                                            // 累计资金费（全部时间）
	CumFundingSinceOpen   string `gorm:"type:numeric"`                                            // 累计资金费（开仓以来）
	CumFundingSinceChange string `gorm:"type:numeric"`                                            // 累计资金费（最近变更以来）
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
