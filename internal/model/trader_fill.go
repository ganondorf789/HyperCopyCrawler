package model

import "time"

// TraderFill 交易员成交记录表（trader_fills）
// 组合唯一索引：(time, hash, oid, tid)
type TraderFill struct {
	ID            uint      `gorm:"primaryKey;comment:主键ID"`
	Address       string    `gorm:"type:varchar(42);not null;index;comment:钱包地址"`
	Coin          string    `gorm:"type:varchar(20);not null;comment:币种"`
	Px            string    `gorm:"type:numeric;not null;comment:成交价"`
	Sz            string    `gorm:"type:numeric;not null;comment:成交量"`
	Side          string    `gorm:"type:varchar(2);not null;comment:买卖方向（A=卖/B=买）"`
	Time          int64     `gorm:"not null;index;uniqueIndex:uidx_fill;comment:成交时间（毫秒时间戳）"`
	StartPosition string    `gorm:"type:numeric;comment:成交前仓位大小"`
	Dir           string    `gorm:"type:varchar(20);comment:操作方向（Open Long/Open Short/Close Long/Close Short）"`
	ClosedPnl     string    `gorm:"type:numeric;comment:平仓盈亏"`
	Hash          string    `gorm:"type:varchar(66);not null;index;uniqueIndex:uidx_fill;comment:交易哈希"`
	Oid           int64     `gorm:"not null;uniqueIndex:uidx_fill;comment:订单ID"`
	Crossed       bool      `gorm:"not null;default:false;comment:是否为全仓模式"`
	Fee           string    `gorm:"type:numeric;comment:手续费"`
	Tid           int64     `gorm:"not null;index;uniqueIndex:uidx_fill;comment:成交ID"`
	Cloid         string    `gorm:"type:varchar(66);comment:客户端订单ID"`
	FeeToken      string    `gorm:"type:varchar(10);comment:手续费计价币种"`
	CreatedAt     time.Time `gorm:"comment:创建时间"`
}
