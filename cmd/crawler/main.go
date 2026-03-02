package main

import (
	"fmt"
	"os"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/crawler"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/logger"
	"go.uber.org/zap"
)

func main() {
	_, cleanup, err := logger.Init("crawler")
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	cfg := config.Load()

	// Connect PostgreSQL
	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		zap.S().Fatalf("postgres: %v", err)
	}

	// Connect Redis
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		zap.S().Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	zap.S().Info("HyperCopyCrawler started")

	c := crawler.New(db)

	// Step 1: 获取排行榜前5000交易员 -> Trader + TraderPerformance
	if err := c.SyncLeaderboard(); err != nil {
		zap.S().Fatalf("sync leaderboard: %v", err)
	}

	// Step 2: 获取每个交易员的 portfolio -> TraderPnlHistory + TraderAccountValue
	if err := c.SyncPortfolios(); err != nil {
		zap.S().Fatalf("sync portfolios: %v", err)
	}

	zap.S().Info("HyperCopyCrawler finished")
}
