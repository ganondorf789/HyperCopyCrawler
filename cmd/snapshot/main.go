package main

import (
	"flag"
	"log"
	"time"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/proxy"
	"github.com/hypercopy/crawler/internal/snapshot"
)

func main() {
	rate := flag.Int("rate", 5, "并发 worker 数量")
	delay := flag.Duration("delay", 500*time.Millisecond, "每个 worker 请求间隔")
	flag.Parse()

	cfg := config.Load()

	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	proxyMgr, err := proxy.NewManager(db)
	if err != nil {
		log.Fatalf("proxy manager: %v", err)
	}
	log.Printf("[main] %d proxies loaded, %d workers, delay %v", proxyMgr.Count(), *rate, *delay)

	s := snapshot.NewSyncer(db, proxyMgr, *rate, *delay)
	s.Run()
}
