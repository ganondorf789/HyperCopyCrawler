package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/fills"
	"github.com/hypercopy/crawler/internal/logger"
	"github.com/hypercopy/crawler/internal/proxy"
	"go.uber.org/zap"
)

func main() {
	workers := flag.Int("workers", 10, "并发 worker 数量")
	delay := flag.Duration("delay", 200*time.Millisecond, "每次 API 请求间隔")
	useProxy := flag.Bool("proxy", false, "是否启用代理池")
	flag.Parse()

	_, cleanup, err := logger.Init("fills")
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

	var proxyMgr *proxy.Manager
	if *useProxy {
		proxyMgr, err = proxy.NewManager(db)
		if err != nil {
			zap.S().Fatalf("proxy manager: %v", err)
		}
		zap.S().Infof("[main] proxy enabled, %d proxies loaded, %d workers", proxyMgr.Count(), *workers)
	} else {
		zap.S().Infof("[main] proxy disabled, %d workers (direct connection)", *workers)
	}

	w := fills.NewWorker(db, proxyMgr, *workers, *delay)
	for round := 1; ; round++ {
		zap.S().Infof("[main] fills sync round %d starting", round)
		if err := w.Run(); err != nil {
			zap.S().Errorf("[main] fills sync round %d error: %v, retrying...", round, err)
			continue
		}
		zap.S().Infof("[main] fills sync round %d finished", round)
	}
}
