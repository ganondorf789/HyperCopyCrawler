package main

import (
	"flag"
	"log"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/database"
	"github.com/hypercopy/crawler/internal/proxy"
	"github.com/hypercopy/crawler/internal/snapshot"
)

func main() {
	rate := flag.Int("rate", 5, "并发 worker 数量")
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
	log.Printf("[main] %d proxies loaded, %d workers", proxyMgr.Count(), *rate)

	s := snapshot.NewSyncer(db, proxyMgr, *rate)
	s.Run()
}
