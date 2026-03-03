package model

import "time"

type UserAppKey struct {
	ID        int64      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    int64      `gorm:"not null;default:0;index:idx_user_app_key_user_id;comment:所属用户ID" json:"user_id"`
	AppID     string     `gorm:"type:varchar(64);not null;uniqueIndex;comment:AppID" json:"app_id"`
	AppSecret string     `gorm:"type:varchar(128);not null;comment:AppSecret" json:"-"`
	Remark    string     `gorm:"type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	ExpireAt  *time.Time `gorm:"index:idx_user_app_key_expire_at;comment:过期时间,NULL表示永不过期" json:"expire_at"`
	Status    int16      `gorm:"type:smallint;not null;default:1;index:idx_user_app_key_status;comment:状态 1:启用 0:禁用" json:"status"`
	CreatedAt time.Time  `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (UserAppKey) TableName() string {
	return "user_app_key"
}
