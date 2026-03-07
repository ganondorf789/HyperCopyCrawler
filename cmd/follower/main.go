package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/follower"
	"github.com/hypercopy/crawler/internal/logger"
	"go.uber.org/zap"
)

func main() {
	serverIP := flag.String("server-ip", "", "本机 IP，用于从 Redis 分配表获取负责的地址列表")
	flag.Parse()

	if *serverIP == "" {
		fmt.Fprintln(os.Stderr, "-server-ip is required")
		os.Exit(1)
	}

	_, cleanup, err := logger.Init("follower")
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

	zap.S().Infof("[main] server-ip=%s", *serverIP)

	f := follower.New(db, rdb, *serverIP)
	f.Run()
}
