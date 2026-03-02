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
	if err := db.AutoMigrate(&model.Trader{}, &model.TraderPerformance{}, &model.TraderAccountValue{}, &model.TraderPnlHistory{}, &model.TraderFill{}, &model.ProxyPool{}, &model.CompletedTrade{}, &model.TraderPosition{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	zap.S().Info("[postgres] connected and migrated successfully")
	return db, nil
}
