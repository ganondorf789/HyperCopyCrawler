package model

import "time"

// Server 服务器管理表（server_management）
type Server struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	IP        string    `gorm:"type:varchar(45);not null;index:idx_server_ip;comment:服务器IP地址" json:"ip"`
	Username  string    `gorm:"type:varchar(255);not null;default:'';comment:用户名" json:"username"`
	Password  string    `gorm:"type:varchar(255);not null;default:'';comment:密码" json:"password"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Server) TableName() string {
	return "server_management"
}
