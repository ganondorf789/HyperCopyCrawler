package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/funding"
	"github.com/hypercopy/crawler/internal/logger"
	"github.com/hypercopy/crawler/internal/proxy"
	"go.uber.org/zap"
)

func main() {
	workers := flag.Int("workers", 10, "并发 worker 数量")
	delay := flag.Duration("delay", 200*time.Millisecond, "每次 API 请求间隔")
	flag.Parse()

	_, cleanup, err := logger.Init("funding")
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	cfg := config.Load()

	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		zap.S().Fatalf("postgres: %v", err)
	}

	proxyMgr, err := proxy.NewManager(db)
	if err != nil {
		zap.S().Fatalf("proxy manager: %v", err)
	}
	zap.S().Infof("[main] %d proxies loaded, %d workers", proxyMgr.Count(), *workers)

	w := funding.NewWorker(db, proxyMgr, *workers, *delay)
	if err := w.Run(); err != nil {
		zap.S().Fatalf("run: %v", err)
	}

	zap.S().Info("[main] funding sync finished")
}
