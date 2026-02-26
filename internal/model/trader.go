package model

import (
	"time"

	"github.com/lib/pq"
)

// Trader 交易员
type Trader struct {
	ID             uint           `gorm:"primaryKey"`
	TwitterName    string         `gorm:"type:varchar(255);not null"`                    // 推特显示名
	Username       string         `gorm:"type:varchar(255);not null;uniqueIndex"`        // 推特用户名
	Address        string         `gorm:"type:varchar(42);not null;uniqueIndex"`         // 钱包地址
	ProfilePicture string         `gorm:"type:text"`                                     // 头像链接
	Labels                  pq.StringArray `gorm:"type:text[];default:'{}'"` // 标签列表
	SnapEffLeverage         string         `gorm:"type:numeric"`             // 有效杠杆
	SnapLongPositionCount   int            `gorm:"default:0"`                // 多头持仓数
	SnapLongPositionValue   string         `gorm:"type:numeric"`             // 多头持仓价值
	SnapMarginUsageRate     string         `gorm:"type:numeric"`             // 保证金使用率
	SnapPerpValue           string         `gorm:"type:numeric"`             // 永续合约价值
	SnapPositionCount       int            `gorm:"default:0"`                // 总持仓数
	SnapPositionValue       string         `gorm:"type:numeric"`             // 总持仓价值
	SnapShortPositionCount  int            `gorm:"default:0"`                // 空头持仓数
	SnapShortPositionValue  string         `gorm:"type:numeric"`             // 空头持仓价值
	SnapSpotValue           string         `gorm:"type:numeric"`             // 现货价值
	SnapTotalMarginUsed     string         `gorm:"type:numeric"`             // 已用保证金
	SnapTotalValue          string         `gorm:"type:numeric"`             // 总价值
	SnapUnrealizedPnl       string         `gorm:"type:numeric"`             // 未实现盈亏
	ShortPnl                string         `gorm:"type:numeric"`             // 空头盈亏
	ShortWinRate            *float64       `gorm:"type:numeric"`             // 空头胜率
	LongPnl                 string         `gorm:"type:numeric"`             // 多头盈亏
	LongWinRate             *float64       `gorm:"type:numeric"`             // 多头胜率
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
