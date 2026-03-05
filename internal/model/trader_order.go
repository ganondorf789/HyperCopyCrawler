package model

import "time"

// TraderOrder 交易员历史委托记录表（trader_orders）
type TraderOrder struct {
	ID               uint      `gorm:"primaryKey;comment:主键ID"`
	Address          string    `gorm:"type:varchar(42);not null;index:idx_tord_addr_ts;uniqueIndex:uidx_order;comment:钱包地址"`
	Coin             string    `gorm:"type:varchar(20);not null;uniqueIndex:uidx_order;comment:币种"`
	Side             string    `gorm:"type:varchar(2);not null;comment:买卖方向（A=卖/B=买）"`
	LimitPx          string    `gorm:"type:numeric;not null;comment:限价"`
	Sz               string    `gorm:"type:numeric;not null;comment:委托量"`
	Oid              int64     `gorm:"not null;index;uniqueIndex:uidx_order;comment:订单ID"`
	Timestamp        int64     `gorm:"not null;index:idx_tord_addr_ts;uniqueIndex:uidx_order;comment:委托时间（毫秒时间戳）"`
	TriggerCondition string    `gorm:"type:varchar(20);not null;default:'N/A';comment:触发条件"`
	IsTrigger        bool      `gorm:"not null;default:false;comment:是否触发订单"`
	TriggerPx        string    `gorm:"type:numeric;comment:触发价"`
	Children         string    `gorm:"type:jsonb;default:'[]';comment:子订单(JSON)"`
	IsPositionTpsl   bool      `gorm:"not null;default:false;comment:是否为仓位止盈止损"`
	ReduceOnly       bool      `gorm:"not null;default:false;comment:是否只减仓"`
	OrderType        string    `gorm:"type:varchar(20);not null;comment:订单类型"`
	OrigSz           string    `gorm:"type:numeric;not null;comment:原始委托量"`
	Tif              string    `gorm:"type:varchar(20);comment:有效期类型"`
	Cloid            *string   `gorm:"type:varchar(66);comment:客户端订单ID"`
	Status           string    `gorm:"type:varchar(20);not null;comment:订单状态"`
	CreatedAt        time.Time `gorm:"comment:创建时间"`
}
