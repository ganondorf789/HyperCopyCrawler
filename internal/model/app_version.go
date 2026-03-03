package model

import "time"

type AppVersion struct {
	ID           int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	Platform     string    `gorm:"type:varchar(32);not null;default:'';index:idx_app_version_platform;comment:平台 ios/android" json:"platform"`
	VersionName  string    `gorm:"type:varchar(32);not null;default:'';comment:版本号 如1.2.0" json:"version_name"`
	VersionCode  int       `gorm:"not null;default:0;comment:版本编码 用于比较大小" json:"version_code"`
	DownloadURL  string    `gorm:"type:varchar(512);not null;default:'';comment:下载地址" json:"download_url"`
	ChangeLog    string    `gorm:"type:text;not null;default:'';comment:更新日志" json:"change_log"`
	ForceUpdate  int16     `gorm:"type:smallint;not null;default:0;comment:是否强制更新 0:否 1:是" json:"force_update"`
	MinVersionCode int     `gorm:"not null;default:0;comment:最低兼容版本编码 低于此版本强制更新" json:"min_version_code"`
	Status       int16     `gorm:"type:smallint;not null;default:1;index:idx_app_version_status;comment:状态 1:已发布 0:未发布" json:"status"`
	PublishedAt  *time.Time `gorm:"comment:发布时间" json:"published_at"`
	CreatedAt    time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (AppVersion) TableName() string {
	return "app_version"
}
