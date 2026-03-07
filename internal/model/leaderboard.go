package model

import "time"

// Leaderboard 窗口常量
const (
	LeaderboardWindowDay     = "day"
	LeaderboardWindowWeek    = "week"
	LeaderboardWindowMonth   = "month"
	LeaderboardWindowAllTime = "allTime"
)

// Leaderboard 排行榜表（leaderboard）
type Leaderboard struct {
	ID           uint      `gorm:"primaryKey;comment:主键ID"`
	Window       string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_leaderboard_window_addr;comment:统计窗口 day/week/month/allTime"`
	EthAddress   string    `gorm:"type:varchar(42);not null;uniqueIndex:idx_leaderboard_window_addr;comment:钱包地址"`
	AccountValue string    `gorm:"type:numeric;not null;default:0;comment:账户价值"`
	Pnl          string    `gorm:"type:numeric;not null;default:0;comment:盈亏"`
	Roi          string    `gorm:"type:numeric;not null;default:0;comment:投资回报率"`
	Vlm          string    `gorm:"type:numeric;not null;default:0;comment:交易量"`
	CreatedAt    time.Time `gorm:"comment:创建时间"`
	UpdatedAt    time.Time `gorm:"comment:更新时间"`
}

func (Leaderboard) TableName() string {
	return "leaderboard"
}
