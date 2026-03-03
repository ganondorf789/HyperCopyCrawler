package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hypercopy/crawler/internal/config"
	"github.com/hypercopy/crawler/internal/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	_, cleanup, err := logger.Init("resetdb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	cfg := config.Load()

	fmt.Printf("⚠  WARNING: This will DROP ALL TABLES in database [%s] at %s:%s\n",
		cfg.Postgres.DBName, cfg.Postgres.Host, cfg.Postgres.Port)
	fmt.Print("Are you sure? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	db, err := gorm.Open(postgres.Open(cfg.Postgres.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		zap.S().Fatalf("postgres: %v", err)
	}

	fmt.Println("Dropping all tables...")

	if err := db.Exec("DROP SCHEMA public CASCADE").Error; err != nil {
		zap.S().Fatalf("failed to drop schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA public").Error; err != nil {
		zap.S().Fatalf("failed to recreate schema: %v", err)
	}

	fmt.Println("Done. All tables have been dropped.")
}
