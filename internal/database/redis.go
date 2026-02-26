package database

import (
	"context"
	"fmt"
	"log"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	log.Println("[redis] connected successfully")
	return rdb, nil
}
