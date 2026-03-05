package model

import (
	"time"

	"github.com/lib/pq"
)

// Trader 交易员信息表（traders）
type Trader struct {
	ID                     uint           `gorm:"primaryKey;comment:主键ID"`
	TwitterName            string         `gorm:"type:varchar(255);default:'';comment:推特显示名"`
	Username               string         `gorm:"type:varchar(255);default:'';comment:推特用户名"`
	Address                string         `gorm:"type:varchar(42);not null;uniqueIndex;comment:钱包地址"`
	ProfilePicture         string         `gorm:"type:text;comment:头像链接"`
	IsHotAddress           bool           `gorm:"default:false;comment:是否热门地址"`
	IsTwitterKOL           bool           `gorm:"default:false;comment:是否推特KOL"`
	Labels                 pq.StringArray `gorm:"type:text[];default:'{}';comment:标签列表"`
	SnapEffLeverage        string         `gorm:"type:numeric;comment:快照-有效杠杆"`
	SnapLongPositionCount  int            `gorm:"default:0;comment:快照-多头持仓数"`
	SnapLongPositionValue  string         `gorm:"type:numeric;comment:快照-多头持仓价值"`
	SnapMarginUsageRate    string         `gorm:"type:numeric;comment:快照-保证金使用率"`
	SnapPerpValue          string         `gorm:"type:numeric;comment:快照-永续合约价值"`
	SnapPositionCount      int            `gorm:"default:0;comment:快照-总持仓数"`
	SnapPositionValue      string         `gorm:"type:numeric;comment:快照-总持仓价值"`
	SnapShortPositionCount int            `gorm:"default:0;comment:快照-空头持仓数"`
	SnapShortPositionValue string         `gorm:"type:numeric;comment:快照-空头持仓价值"`
	SnapSpotValue          string         `gorm:"type:numeric;comment:快照-现货价值"`
	SnapTotalMarginUsed    string         `gorm:"type:numeric;comment:快照-已用保证金"`
	SnapTotalValue         string         `gorm:"type:numeric;comment:快照-总价值"`
	SnapUnrealizedPnl      string         `gorm:"type:numeric;comment:快照-未实现盈亏"`
	ShortPnl               string         `gorm:"type:numeric;comment:空头盈亏"`
	ShortWinRate           *float64       `gorm:"type:numeric;comment:空头胜率"`
	LongPnl                string         `gorm:"type:numeric;comment:多头盈亏"`
	LongWinRate            *float64       `gorm:"type:numeric;comment:多头胜率"`
	TotalPnl               string         `gorm:"type:numeric;comment:总盈亏"`
	CreatedAt              time.Time      `gorm:"comment:创建时间"`
	UpdatedAt              time.Time      `gorm:"comment:更新时间"`
}
