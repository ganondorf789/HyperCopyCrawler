package main

import (
	"log"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/model"
)

func main() {
	cfg := config.Load()

	// Connect PostgreSQL
	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&model.CrawlTask{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Connect Redis
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	log.Println("HyperCopyCrawler started")

	// TODO: 在这里添加爬虫逻辑
	_ = db
}
