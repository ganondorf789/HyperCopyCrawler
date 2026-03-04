package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CoinPosition 持仓币种明细
type CoinPosition struct {
	Coin string `json:"coin"`
	Szi  string `json:"szi"`
}

// CoinPositions 持仓币种列表，支持 GORM jsonb 读写
type CoinPositions []CoinPosition

func (p CoinPositions) Value() (driver.Value, error) {
	if p == nil {
		return "[]", nil
	}
	b, err := json.Marshal(p)
	return string(b), err
}

func (p *CoinPositions) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}

// TraderCoinHolding 交易员当前持仓币种表（trader_coin_holding）
type TraderCoinHolding struct {
	ID        uint          `gorm:"primaryKey;comment:主键ID"`
	Address   string        `gorm:"type:varchar(42);not null;uniqueIndex;comment:钱包地址"`
	Positions CoinPositions `gorm:"type:jsonb;default:'[]';comment:持仓币种列表" json:"positions"`
	CreatedAt time.Time     `gorm:"comment:创建时间"`
	UpdatedAt time.Time     `gorm:"comment:更新时间"`
}

func (TraderCoinHolding) TableName() string {
	return "trader_coin_holding"
}
