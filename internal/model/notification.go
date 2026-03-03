package model

import "time"

type Notification struct {
	ID        int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    int64     `gorm:"not null;default:0;index:idx_notification_user_id;comment:所属用户ID,0表示公共通知" json:"user_id"`
	Category  string    `gorm:"type:varchar(32);not null;default:'';index:idx_notification_category;comment:通知类型 public/copy_trading/whale/track/market" json:"category"`
	Title     string    `gorm:"type:varchar(255);not null;default:'';comment:通知标题" json:"title"`
	Content   string    `gorm:"type:text;not null;default:'';comment:通知内容" json:"content"`
	RefID     int64     `gorm:"not null;default:0;comment:关联业务ID,0表示无关联" json:"ref_id"`
	RefType   string    `gorm:"type:varchar(32);not null;default:'';comment:关联业务类型 copy_trading/track_wallet/position" json:"ref_type"`
	Level     int16     `gorm:"type:smallint;not null;default:0;comment:通知级别 0:普通 1:重要 2:紧急" json:"level"`
	Status    int16     `gorm:"type:smallint;not null;default:1;comment:状态 1:正常 0:已撤回" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:now();index:idx_notification_created_at;comment:创建时间" json:"created_at"`
}

func (Notification) TableName() string {
	return "notification"
}

type NotificationRead struct {
	ID             int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID         int64     `gorm:"not null;uniqueIndex:uk_notification_read_user_noti;comment:用户ID" json:"user_id"`
	NotificationID int64     `gorm:"not null;uniqueIndex:uk_notification_read_user_noti;comment:通知ID" json:"notification_id"`
	ReadAt         time.Time `gorm:"not null;default:now();comment:已读时间" json:"read_at"`
}

func (NotificationRead) TableName() string {
	return "notification_read"
}
