package model

import "time"

type CronTask struct {
	ID          int64      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string     `gorm:"type:varchar(128);not null;uniqueIndex;comment:任务名称" json:"name"`
	CronExpr    string     `gorm:"type:varchar(64);not null;default:'';comment:Cron表达式" json:"cron_expr"`
	TaskType    string     `gorm:"type:varchar(64);not null;default:'';index:idx_cron_task_type;comment:任务类型 sync_leaderboard/sync_positions/sync_fills/copy_trade/track_wallet/market_alert" json:"task_type"`
	Params      string     `gorm:"type:text;not null;default:'';comment:任务参数(JSON)" json:"params"`
	LastRunAt   *time.Time `gorm:"comment:上次执行时间" json:"last_run_at"`
	LastRunCost int64      `gorm:"not null;default:0;comment:上次执行耗时(毫秒)" json:"last_run_cost"`
	LastError   string     `gorm:"type:text;not null;default:'';comment:上次执行错误信息" json:"last_error"`
	RunCount    int64      `gorm:"not null;default:0;comment:累计执行次数" json:"run_count"`
	Status      int16      `gorm:"type:smallint;not null;default:1;index:idx_cron_task_status;comment:状态 1:启用 0:停用" json:"status"`
	Remark      string     `gorm:"type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	CreatedAt   time.Time  `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (CronTask) TableName() string {
	return "cron_task"
}
