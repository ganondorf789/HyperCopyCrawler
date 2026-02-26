package main

import (
	"flag"
	"log"
	"time"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/fills"
	"github.com/hypercopy/crawler/internal/proxy"
)

func main() {
	workers := flag.Int("workers", 10, "并发 worker 数量")
	delay := flag.Duration("delay", 200*time.Millisecond, "每次 API 请求间隔")
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
	log.Printf("[main] %d proxies loaded, %d workers", proxyMgr.Count(), *workers)

	w := fills.NewWorker(db, proxyMgr, *workers, *delay)
	if err := w.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}

	log.Println("[main] fills sync finished")
}
