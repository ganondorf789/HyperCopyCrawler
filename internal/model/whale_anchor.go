package model

import "time"

type WhaleAnchor struct {
	ID             int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Symbol         string    `gorm:"type:varchar(64);not null;uniqueIndex;comment:交易对符号" json:"symbol"`
	Volume24h      string    `gorm:"type:numeric(30,8);not null;default:0;comment:24h成交量(USD)" json:"volume_24h"`
	OpenInterest   string    `gorm:"type:numeric(30,8);not null;default:0;comment:当前未平仓合约量(USD)" json:"open_interest"`
	Depth1pct      string    `gorm:"type:numeric(30,8);not null;default:0;comment:1%盘口深度(USD)" json:"depth_1pct"`
	ValVolume      string    `gorm:"type:numeric(30,8);not null;default:0;comment:0.4% x 24h Volume" json:"val_volume"`
	ValOI          string    `gorm:"type:numeric(30,8);not null;default:0;comment:1% x OI" json:"val_oi"`
	ValDepth       string    `gorm:"type:numeric(30,8);not null;default:0;comment:30% x 1% Depth" json:"val_depth"`
	WhaleThreshold string    `gorm:"type:numeric(30,8);not null;default:0;comment:巨鲸仓位阈值 max(val_volume,val_oi,val_depth)" json:"whale_threshold"`
	CreatedAt      time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (WhaleAnchor) TableName() string {
	return "whale_anchor"
}
