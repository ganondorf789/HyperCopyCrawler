package model

import "time"

// HotCoin 热门币种表（hot_coin）
type HotCoin struct {
	ID        int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Coin      string    `gorm:"type:varchar(32);not null;uniqueIndex;comment:币种名称" json:"coin"`
	CreatedAt time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (HotCoin) TableName() string {
	return "hot_coin"
}
