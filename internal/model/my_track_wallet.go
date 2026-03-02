package model

import "time"

type MyTrackWallet struct {
	ID           int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID       int64     `gorm:"not null;default:0;index:idx_my_track_wallet_user_id;comment:所属用户ID" json:"user_id"`
	Wallet       string    `gorm:"type:varchar(255);not null;default:'';comment:跟踪的钱包地址" json:"wallet"`
	Remark       string    `gorm:"type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	EnableNotify int16     `gorm:"type:smallint;not null;default:0;comment:是否开启通知 0:关 1:开" json:"enable_notify"`
	NotifyAction string    `gorm:"type:varchar(64);not null;default:'';comment:通知动作 1:开仓 2:平仓 3:加仓 4:减仓" json:"notify_action"`
	Lang         string    `gorm:"type:varchar(16);not null;default:'zh';comment:语言" json:"lang"`
	Status       int16     `gorm:"type:smallint;not null;default:1;index:idx_my_track_wallet_status;comment:状态 1:正常 0:禁用" json:"status"`
	CreatedAt    time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (MyTrackWallet) TableName() string {
	return "my_track_wallet"
}
