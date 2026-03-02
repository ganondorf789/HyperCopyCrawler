package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/logger"
	"github.com/hypercopy/crawler/internal/proxy"
	"github.com/hypercopy/crawler/internal/snapshot"
	"go.uber.org/zap"
)

func main() {
	rate := flag.Int("rate", 5, "并发 worker 数量")
	flag.Parse()

	_, cleanup, err := logger.Init("snapshot")
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
	zap.S().Infof("[main] %d proxies loaded, %d workers", proxyMgr.Count(), *rate)

	s := snapshot.NewSyncer(db, proxyMgr, *rate)
	s.Run()
}
