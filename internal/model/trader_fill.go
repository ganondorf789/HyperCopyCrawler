package model

import "time"

// TraderFill 交易员成交记录
type TraderFill struct {
	ID            uint    `gorm:"primaryKey"`
	Address       string  `gorm:"type:varchar(42);not null;index"`          // 钱包地址
	Coin          string  `gorm:"type:varchar(20);not null"`                // 币种
	Px            string  `gorm:"type:numeric;not null"`                    // 成交价
	Sz            string  `gorm:"type:numeric;not null"`                    // 成交量
	Side          string  `gorm:"type:varchar(2);not null"`                 // A/B
	Time          int64   `gorm:"not null;index"`                           // 毫秒时间戳
	StartPosition string  `gorm:"type:numeric"`                             // 开仓前仓位
	Dir           string  `gorm:"type:varchar(20)"`                         // Open Long/Open Short/Close Long/Close Short
	ClosedPnl     string  `gorm:"type:numeric"`                             // 平仓盈亏
	Hash          string  `gorm:"type:varchar(66);not null;uniqueIndex"`    // 交易哈希
	Oid           int64   `gorm:"not null"`                                 // 订单ID
	Crossed       bool    `gorm:"not null;default:false"`                   // 是否crossed
	Fee           string  `gorm:"type:numeric"`                             // 手续费
	Tid           int64   `gorm:"not null;uniqueIndex"`                     // 成交ID
	Cloid         string  `gorm:"type:varchar(66)"`                         // 客户端订单ID
	FeeToken      string  `gorm:"type:varchar(10)"`                         // 手续费币种
	CreatedAt     time.Time
}
