package model

import "time"

type Admin struct {
	ID        int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Username  string    `gorm:"type:varchar(64);not null;uniqueIndex;comment:用户名" json:"username"`
	Password  string    `gorm:"type:varchar(128);not null;comment:密码" json:"-"`
	Realname  string    `gorm:"type:varchar(64);not null;default:'';comment:真实姓名" json:"realname"`
	Role      string    `gorm:"type:varchar(32);not null;default:'admin';comment:角色 admin/super_admin" json:"role"`
	Status    int16     `gorm:"type:smallint;not null;default:1;index:idx_admin_status;comment:状态 1:正常 0:禁用" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (Admin) TableName() string {
	return "admin"
}
