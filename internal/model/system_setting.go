package model

import "time"

// SystemSetting 系统设置表（system_setting）
type SystemSetting struct {
	ID                     uint      `gorm:"primaryKey;comment:主键ID"`
	MarketMinutes          int       `gorm:"not null;default:5;comment:行情监控时间窗口（分钟）"`
	MarketNewPositionCount int       `gorm:"not null;default:3;comment:时间窗口内新仓位数量阈值"`
	LimitTradingWallet     int       `gorm:"not null;default:0;comment:交易钱包数量限制（0=不限制）"`
	LimitCopyTrading       int       `gorm:"not null;default:0;comment:跟单交易数量限制（0=不限制）"`
	LimitWatchedAddress    int       `gorm:"not null;default:0;comment:监控地址数量限制（0=不限制）"`
	CreatedAt              time.Time `gorm:"comment:创建时间"`
	UpdatedAt              time.Time `gorm:"comment:更新时间"`
}

func (SystemSetting) TableName() string {
	return "system_setting"
}
