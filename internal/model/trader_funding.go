package model

import "time"

// TraderFunding 交易员资金费记录表（trader_fundings）
type TraderFunding struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID"`
	Address     string    `gorm:"type:varchar(42);not null;index:idx_tf_addr_time;uniqueIndex:uidx_funding;comment:钱包地址"`
	Time        int64     `gorm:"not null;index:idx_tf_addr_time;uniqueIndex:uidx_funding;comment:资金费时间（毫秒时间戳）"`
	Hash        string    `gorm:"type:varchar(66);not null;index;comment:交易哈希"`
	Coin        string    `gorm:"type:varchar(20);not null;uniqueIndex:uidx_funding;comment:币种"`
	Usdc        string    `gorm:"type:numeric;not null;comment:USDC金额（正=收入，负=支出）"`
	Szi         string    `gorm:"type:numeric;not null;comment:持仓大小"`
	FundingRate string    `gorm:"type:numeric;not null;comment:资金费率"`
	CreatedAt   time.Time `gorm:"comment:创建时间"`
}
