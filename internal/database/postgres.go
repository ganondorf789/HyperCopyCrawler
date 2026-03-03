package database

import (
	"fmt"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewPostgres(cfg config.PostgresConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&model.Trader{},
		&model.TraderPerformance{},
		&model.TraderAccountValue{},
		&model.TraderPnlHistory{},
		&model.TraderFill{},
		&model.TraderFunding{},
		&model.TraderOrder{},
		&model.ProxyPool{},
		&model.CompletedTrade{},
		&model.TraderPosition{},
		&model.User{},
		&model.Admin{},
		&model.CopyTrading{},
		&model.Wallet{},
		&model.MyTrackWallet{},
		&model.UserPosition{},
		&model.Membership{},
		&model.Notification{},
		&model.NotificationRead{},
		&model.CronTask{},
		&model.AppVersion{},
		&model.WhaleAnchor{},
		&model.UserAppKey{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 为各表添加数据库级别注释
	tableComments := map[string]string{
		"traders":               "交易员信息表",
		"trader_performances":   "交易员绩效表（按时间窗口聚合）",
		"trader_account_values": "交易员账户价值历史表",
		"trader_pnl_histories":  "交易员盈亏历史表",
		"trader_fills":          "交易员成交记录表",
		"trader_fundings":       "交易员资金费记录表",
		"trader_orders":         "交易员历史委托记录表",
		"proxy_pools":           "代理池表",
		"completed_trades":      "已完成交易表（由 fills 聚合而来）",
		"trader_positions":      "交易员当前持仓表",
		"user":                  "用户表",
		"admin":                 "后台管理员表",
		"copy_trading":          "跟单交易配置表",
		"wallet":                "钱包表",
		"my_track_wallet":       "跟踪钱包表",
		"position":              "持仓表",
		"membership":            "会员表",
		"notification":          "通知表",
		"notification_read":     "通知已读记录表",
		"cron_task":             "定时任务表",
		"app_version":           "APP版本管理表",
		"whale_anchor":          "巨鲸锚点表",
		"user_app_key":          "用户AppID/AppSecret管理表",
	}
	for table, comment := range tableComments {
		if err := db.Exec("COMMENT ON TABLE " + table + " IS '" + comment + "'").Error; err != nil {
			zap.S().Warnf("[postgres] failed to set comment on table %s: %v", table, err)
		}
	}

	zap.S().Info("[postgres] connected and migrated successfully")
	return db, nil
}
