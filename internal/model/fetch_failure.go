package model

import "time"

// FetchFailure 数据获取失败表（fetch_failures）
// 当最细粒度窗口超限或429限频重试耗尽时，记录失败信息
// Type 取值：fills、orders、funding
// Reason 取值：exceeds_limit（窗口超限）、rate_limited（429限频）
type FetchFailure struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID"`
	Type        string    `gorm:"type:varchar(20);not null;index;comment:数据类型（fills/orders/funding）"`
	Reason      string    `gorm:"type:varchar(30);not null;default:'exceeds_limit';comment:失败原因（exceeds_limit/rate_limited）"`
	Address     string    `gorm:"type:varchar(42);not null;index;comment:钱包地址"`
	StartMs     int64     `gorm:"not null;comment:失败时间窗口开始（毫秒时间戳）"`
	EndMs       int64     `gorm:"not null;comment:失败时间窗口结束（毫秒时间戳）"`
	RecordCount int       `gorm:"not null;comment:该窗口返回的记录数"`
	CreatedAt   time.Time `gorm:"comment:创建时间"`
}
