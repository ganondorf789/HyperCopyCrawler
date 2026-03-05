package model

import "time"

// OrderFetchFailure 委托记录获取失败表（order_fetch_failures）
// 当30秒窗口仍然超过2000条限制时，记录失败信息
type OrderFetchFailure struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID"`
	Address    string    `gorm:"type:varchar(42);not null;index;comment:钱包地址"`
	StartMs    int64     `gorm:"not null;comment:失败时间窗口开始（毫秒时间戳）"`
	EndMs      int64     `gorm:"not null;comment:失败时间窗口结束（毫秒时间戳）"`
	OrderCount int       `gorm:"not null;comment:该窗口返回的记录数"`
	CreatedAt  time.Time `gorm:"comment:创建时间"`
}
