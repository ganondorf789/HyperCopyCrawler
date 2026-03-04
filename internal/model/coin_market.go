package model

import "time"

// CoinMarket 币种行情数据表（coin_market）
type CoinMarket struct {
	ID              int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Coin            string    `gorm:"type:varchar(32);not null;uniqueIndex;comment:币种名称" json:"coin"`
	Price           string    `gorm:"type:numeric(30,10);not null;default:0;comment:当前价格" json:"price"`
	Change24h       string    `gorm:"type:numeric(30,10);not null;default:0;comment:24h价格变动" json:"change_24h"`
	ChangePercent24h string   `gorm:"type:numeric(20,8);not null;default:0;comment:24h价格变动百分比" json:"change_percent_24h"`
	Open24h         string    `gorm:"type:numeric(30,10);not null;default:0;comment:24h开盘价" json:"open_24h"`
	Close24h        string    `gorm:"type:numeric(30,10);not null;default:0;comment:24h收盘价" json:"close_24h"`
	High24h         string    `gorm:"type:numeric(30,10);not null;default:0;comment:24h最高价" json:"high_24h"`
	Low24h          string    `gorm:"type:numeric(30,10);not null;default:0;comment:24h最低价" json:"low_24h"`
	Volume24h       string    `gorm:"type:numeric(30,4);not null;default:0;comment:24h成交量" json:"volume_24h"`
	QuoteVolume24h  string    `gorm:"type:numeric(30,4);not null;default:0;comment:24h计价成交额" json:"quote_volume_24h"`
	Funding         string    `gorm:"type:numeric(20,10);not null;default:0;comment:资金费率" json:"funding"`
	OpenInterest    string    `gorm:"type:numeric(30,4);not null;default:0;comment:未平仓合约量" json:"open_interest"`
	CreatedAt       time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (CoinMarket) TableName() string {
	return "coin_market"
}
