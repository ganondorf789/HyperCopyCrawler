package model

import "time"

type User struct {
	ID        int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Username  string    `gorm:"type:varchar(64);not null;uniqueIndex;comment:用户名" json:"username"`
	Password  string    `gorm:"type:varchar(128);not null;comment:密码" json:"-"`
	Nickname  string    `gorm:"type:varchar(64);not null;default:'';comment:昵称" json:"nickname"`
	Avatar    string    `gorm:"type:varchar(255);not null;default:'';comment:头像" json:"avatar"`
	Email     string    `gorm:"type:varchar(128);not null;default:'';comment:邮箱" json:"email"`
	Phone     string    `gorm:"type:varchar(32);not null;default:'';comment:手机号" json:"phone"`
	Status    int16     `gorm:"type:smallint;not null;default:1;index:idx_user_status;comment:状态 1:正常 0:禁用" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (User) TableName() string {
	return "user"
}
