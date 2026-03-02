package model

import "time"

// ProxyPool 代理池表（proxy_pools）
type ProxyPool struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID"`
	Host      string    `gorm:"type:varchar(255);not null;comment:代理主机地址"`
	Port      string    `gorm:"type:varchar(10);not null;comment:代理端口"`
	Username  string    `gorm:"type:varchar(255);comment:认证用户名"`
	Password  string    `gorm:"type:varchar(255);comment:认证密码"`
	Status    int       `gorm:"type:smallint;default:1;comment:状态（1=启用 0=禁用）"`
	Remark    string    `gorm:"type:varchar(255);comment:备注"`
	CreatedAt time.Time `gorm:"comment:创建时间"`
	UpdatedAt time.Time `gorm:"comment:更新时间"`
}
