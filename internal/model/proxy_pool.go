package model

import "time"

// ProxyPool 代理池
type ProxyPool struct {
	ID        uint   `gorm:"primaryKey"`
	Host      string `gorm:"type:varchar(255);not null"` // 代理主机地址
	Port      string `gorm:"type:varchar(10);not null"`  // 代理端口
	Username  string `gorm:"type:varchar(255)"`          // 认证用户名
	Password  string `gorm:"type:varchar(255)"`          // 认证密码
	Status    int    `gorm:"type:smallint;default:1"`    // 状态: 1=启用 0=禁用
	Remark    string `gorm:"type:varchar(255)"`          // 备注
	CreatedAt time.Time
	UpdatedAt time.Time
}
