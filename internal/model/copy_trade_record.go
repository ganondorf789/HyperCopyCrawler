package model

import "time"

// CopyTradeRecord 跟单记录表（copy_trade_record）
//
// ExecuteStatus 执行状态：
//
//	0=待执行 1=执行成功 2=执行失败 3=已跳过
//
// OrderStatus 订单状态（同步自交易所）：
//
//	open=挂单中 filled=已成交 canceled=已取消 triggered=已触发
type CopyTradeRecord struct {
	ID            int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID        int64     `gorm:"not null;default:0;index:idx_ctr_user;comment:所属用户ID" json:"user_id"`
	Address       string    `gorm:"type:varchar(42);not null;default:'';comment:钱包地址" json:"address"`
	Coin          string    `gorm:"type:varchar(32);not null;default:'';comment:币种" json:"coin"`
	Direction     string    `gorm:"type:varchar(16);not null;default:'';comment:方向（Open Long/Open Short/Close Long/Close Short）" json:"direction"`
	Size          string    `gorm:"type:numeric(30,8);not null;default:0;comment:成交规模（张数）" json:"size"`
	Price         string    `gorm:"type:numeric(30,10);not null;default:0;comment:成交价格" json:"price"`
	ClosedPnl     string    `gorm:"type:numeric(30,8);not null;default:0;comment:已实现盈亏（USD）" json:"closed_pnl"`
	ExecuteStatus int       `gorm:"not null;default:0;index:idx_ctr_exec;comment:执行状态 0:待执行 1:成功 2:失败 3:跳过" json:"execute_status"`
	OrderStatus   string    `gorm:"type:varchar(32);not null;default:'';comment:订单状态 open/filled/canceled/triggered" json:"order_status"`
	ErrorMsg      string    `gorm:"type:text;not null;default:'';comment:执行失败原因" json:"error_msg"`
	TradeTime     time.Time `gorm:"not null;index:idx_ctr_user;comment:触发交易时间（源头成交时间）" json:"trade_time"`
	CreatedAt     time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (CopyTradeRecord) TableName() string {
	return "copy_trade_record"
}
