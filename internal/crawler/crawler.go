package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Crawler struct {
	db     *gorm.DB
	client *hyperliquid.Client
}

func New(db *gorm.DB) *Crawler {
	return &Crawler{
		db:     db,
		client: hyperliquid.NewClient(),
	}
}

// SyncLeaderboard 获取排行榜前5000交易员，保存地址到Trader表，windowPerformances保存到TraderPerformance表
func (c *Crawler) SyncLeaderboard() error {
	log.Println("[crawler] fetching leaderboard...")
	resp, err := c.client.FetchLeaderboard()
	if err != nil {
		return fmt.Errorf("fetch leaderboard: %w", err)
	}

	rows := resp.LeaderboardRows
	log.Printf("[crawler] got %d leaderboard rows", len(rows))

	// 按30D交易量排序，取前5000
	sort.Slice(rows, func(i, j int) bool {
		return monthVlm(rows[i]) > monthVlm(rows[j])
	})
	if len(rows) > 5000 {
		rows = rows[:5000]
	}
	log.Printf("[crawler] top %d traders by 30D volume", len(rows))

	// 批量保存
	batchSize := 200
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]

		// 1. 保存地址到 Trader 表（只写 address，其他字段保持不变）
		traders := make([]model.Trader, 0, len(batch))
		for _, row := range batch {
			traders = append(traders, model.Trader{
				Address: row.EthAddress,
			})
		}
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}},
			DoNothing: true,
		}).Create(&traders).Error; err != nil {
			return fmt.Errorf("upsert traders batch %d: %w", i/batchSize, err)
		}

		// 2. 保存 windowPerformances 到 TraderPerformance 表
		var perfs []model.TraderPerformance
		for _, row := range batch {
			for _, wp := range row.WindowPerformances {
				window, data, err := wp.Parse()
				if err != nil {
					log.Printf("[crawler] skip bad windowPerformance for %s: %v", row.EthAddress, err)
					continue
				}
				perfs = append(perfs, model.TraderPerformance{
					Address: row.EthAddress,
					Window:  window,
					Pnl:     data.Pnl,
					Roi:     data.Roi,
					Vlm:     data.Vlm,
				})
			}
		}
		if len(perfs) > 0 {
			if err := c.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}, {Name: "window"}},
				DoUpdates: clause.AssignmentColumns([]string{"pnl", "roi", "vlm", "updated_at"}),
			}).Create(&perfs).Error; err != nil {
				return fmt.Errorf("upsert performances batch %d: %w", i/batchSize, err)
			}
		}

		log.Printf("[crawler] saved batch %d/%d (%d traders)", i/batchSize+1, (len(rows)+batchSize-1)/batchSize, len(batch))
	}

	log.Println("[crawler] leaderboard sync done")
	return nil
}

// SyncPortfolios 获取所有Trader的 accountValueHistory 和 pnlHistory
func (c *Crawler) SyncPortfolios() error {
	var traders []model.Trader
	if err := c.db.Select("address").Find(&traders).Error; err != nil {
		return fmt.Errorf("load traders: %w", err)
	}
	log.Printf("[crawler] syncing portfolios for %d traders", len(traders))

	for i, trader := range traders {
		if err := c.syncOnePortfolio(trader.Address); err != nil {
			log.Printf("[crawler] portfolio error for %s: %v", trader.Address, err)
			continue
		}
		if (i+1)%100 == 0 {
			log.Printf("[crawler] portfolio progress: %d/%d", i+1, len(traders))
		}
		// 限速：避免被封
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("[crawler] portfolio sync done")
	return nil
}

func (c *Crawler) syncOnePortfolio(address string) error {
	resp, err := c.client.FetchPortfolio(address)
	if err != nil {
		return err
	}

	// 保存 pnlHistory -> trader_pnl_histories
	if len(resp.PnlHistory) > 0 {
		historyJSON, _ := json.Marshal(resp.PnlHistory)
		record := model.TraderPnlHistory{
			Address: address,
			Window:  "allTime",
			History: historyJSON,
		}
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}, {Name: "window"}},
			DoUpdates: clause.AssignmentColumns([]string{"history", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return fmt.Errorf("upsert pnl history: %w", err)
		}
	}

	// 保存 accountValueHistory -> trader_account_values
	if len(resp.AccountValueHistory) > 0 {
		historyJSON, _ := json.Marshal(resp.AccountValueHistory)
		record := model.TraderAccountValue{
			Address: address,
			Window:  "allTime",
			History: historyJSON,
		}
		if err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}, {Name: "window"}},
			DoUpdates: clause.AssignmentColumns([]string{"history", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return fmt.Errorf("upsert account value: %w", err)
		}
	}

	return nil
}

// monthVlm 提取30D交易量用于排序
func monthVlm(row hyperliquid.LeaderboardRow) float64 {
	for _, wp := range row.WindowPerformances {
		window, data, err := wp.Parse()
		if err != nil {
			continue
		}
		if window == "month" {
			var v float64
			fmt.Sscanf(data.Vlm, "%f", &v)
			return v
		}
	}
	return 0
}
