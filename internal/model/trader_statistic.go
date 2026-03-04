package model

import "time"

// TraderStatistic 交易员统计指标表（trader_statistics）
type TraderStatistic struct {
	ID                 uint      `gorm:"primaryKey;comment:主键ID"`
	Address            string    `gorm:"type:varchar(42);not null;uniqueIndex:idx_stat_addr_window;comment:钱包地址"`
	Window             string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_stat_addr_window;comment:统计窗口（day/week/month/allTime）"`
	Sharpe             string    `gorm:"type:numeric;comment:夏普比率"`
	Drawdown           string    `gorm:"type:numeric;comment:最大回撤"`
	PositionCount      string    `gorm:"type:numeric;comment:持仓数"`
	TotalValue         string    `gorm:"type:numeric;comment:账户总价值"`
	PerpValue          string    `gorm:"type:numeric;comment:永续合约总价值"`
	PositionValue      string    `gorm:"type:numeric;comment:持仓价值"`
	LongPositionValue  string    `gorm:"type:numeric;comment:多仓仓位价值"`
	ShortPositionValue string    `gorm:"type:numeric;comment:空仓仓位价值"`
	MarginUsage        string    `gorm:"type:numeric;comment:保证金使用率"`
	UsedMargin         string    `gorm:"type:numeric;comment:已用保证金"`
	ProfitCount        string    `gorm:"type:numeric;comment:盈利次数"`
	WinRate            string    `gorm:"type:numeric;comment:胜率"`
	TotalPnl           string    `gorm:"type:numeric;comment:总盈亏"`
	LongCount          string    `gorm:"type:numeric;comment:多仓数"`
	LongRealizedPnl    string    `gorm:"type:numeric;comment:多仓已实现盈亏"`
	LongWinRate        string    `gorm:"type:numeric;comment:多仓胜率"`
	ShortCount         string    `gorm:"type:numeric;comment:空仓数"`
	ShortRealizedPnl   string    `gorm:"type:numeric;comment:空仓已实现盈亏"`
	ShortWinRate       string    `gorm:"type:numeric;comment:空仓胜率"`
	UnrealizedPnl      string    `gorm:"type:numeric;comment:未实现盈亏"`
	AvgLeverage        string    `gorm:"type:numeric;comment:平均杠杆"`
	CreatedAt          time.Time `gorm:"comment:创建时间"`
	UpdatedAt          time.Time `gorm:"comment:更新时间"`
}
