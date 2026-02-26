package main

import (
	"log"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/crawler"
	"github.com/hypercopy/crawler/internal/database"
)

func main() {
	cfg := config.Load()

	// Connect PostgreSQL
	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	// Connect Redis
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	log.Println("HyperCopyCrawler started")

	c := crawler.New(db)

	// Step 1: 获取排行榜前5000交易员 -> Trader + TraderPerformance
	if err := c.SyncLeaderboard(); err != nil {
		log.Fatalf("sync leaderboard: %v", err)
	}

	// Step 2: 获取每个交易员的 portfolio -> TraderPnlHistory + TraderAccountValue
	if err := c.SyncPortfolios(); err != nil {
		log.Fatalf("sync portfolios: %v", err)
	}

	log.Println("HyperCopyCrawler finished")
}
