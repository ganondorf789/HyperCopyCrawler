package model

import "time"

type UserPosition struct {
	ID               int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	User             string    `gorm:"column:user;type:varchar(255);not null;default:'';index:idx_position_user;comment:用户钱包地址" json:"user"`
	Symbol           string    `gorm:"type:varchar(64);not null;default:'';index:idx_position_symbol;comment:交易对符号" json:"symbol"`
	PositionSize     string    `gorm:"type:numeric(30,8);not null;default:0;comment:持仓数量（负数为空头）" json:"position_size"`
	EntryPrice       string    `gorm:"type:numeric(30,8);not null;default:0;comment:开仓均价" json:"entry_price"`
	MarkPrice        string    `gorm:"type:numeric(30,8);not null;default:0;comment:标记价格" json:"mark_price"`
	LiqPrice         string    `gorm:"type:numeric(30,8);not null;default:0;comment:强平价格" json:"liq_price"`
	Leverage         int       `gorm:"not null;default:1;comment:杠杆倍数" json:"leverage"`
	MarginBalance    string    `gorm:"type:numeric(30,8);not null;default:0;comment:保证金余额" json:"margin_balance"`
	PositionValueUsd string    `gorm:"type:numeric(30,8);not null;default:0;comment:持仓价值(USD)" json:"position_value_usd"`
	UnrealizedPnl    string    `gorm:"type:numeric(30,8);not null;default:0;comment:未实现盈亏" json:"unrealized_pnl"`
	FundingFee       string    `gorm:"type:numeric(30,8);not null;default:0;comment:资金费用" json:"funding_fee"`
	MarginMode       string    `gorm:"type:varchar(16);not null;default:'cross';comment:保证金模式 cross/isolated" json:"margin_mode"`
	Labels           string    `gorm:"type:text;not null;default:'';comment:标签,逗号分隔" json:"labels"`
	CreatedAt        time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (UserPosition) TableName() string {
	return "position"
}
