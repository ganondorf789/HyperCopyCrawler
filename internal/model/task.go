package model

import "time"

// CrawlTask 爬虫任务
type CrawlTask struct {
	ID        uint      `gorm:"primaryKey"`
	URL       string    `gorm:"type:text;not null"`
	Status    string    `gorm:"type:varchar(20);default:'pending'"` // pending, running, done, failed
	Result    string    `gorm:"type:text"`
	Error     string    `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
