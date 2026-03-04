package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/logger"
	"github.com/hypercopy/crawler/internal/proxy"
	"github.com/hypercopy/crawler/internal/watcher"
	"go.uber.org/zap"
)

func main() {
	rate := flag.Int("rate", 10, "每秒请求 clearinghouseState 的次数")
	offset := flag.Int("offset", 0, "跳过排行榜前 N 名交易员")
	limit := flag.Int("limit", 0, "监控的交易员数量（0 表示不限制）")
	flag.Parse()

	_, cleanup, err := logger.Init("watcher")
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

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		zap.S().Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	proxyMgr, err := proxy.NewManager(db)
	if err != nil {
		zap.S().Fatalf("proxy manager: %v", err)
	}
	zap.S().Infof("[main] %d proxies loaded, rate=%d/s, offset=%d, limit=%d",
		proxyMgr.Count(), *rate, *offset, *limit)

	w := watcher.New(db, rdb, proxyMgr, *rate, *offset, *limit)
	w.Run()
}
