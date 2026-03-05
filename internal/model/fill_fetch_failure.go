package model

import "time"

// FillFetchFailure 成交记录获取失败表（fill_fetch_failures）
// 当30秒窗口仍然超过2000条限制时，记录失败信息
type FillFetchFailure struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID"`
	Address   string    `gorm:"type:varchar(42);not null;index;comment:钱包地址"`
	StartMs   int64     `gorm:"not null;comment:失败时间窗口开始（毫秒时间戳）"`
	EndMs     int64     `gorm:"not null;comment:失败时间窗口结束（毫秒时间戳）"`
	FillCount int       `gorm:"not null;comment:该窗口返回的记录数"`
	CreatedAt time.Time `gorm:"comment:创建时间"`
}
