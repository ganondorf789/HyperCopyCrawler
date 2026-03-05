package crawler

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/proxy"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Crawler struct {
	db       *gorm.DB
	proxyMgr *proxy.Manager
	workers  int
	delay    time.Duration
}

func New(db *gorm.DB, proxyMgr *proxy.Manager, workers int, delay time.Duration) *Crawler {
	return &Crawler{
		db:       db,
		proxyMgr: proxyMgr,
		workers:  workers,
		delay:    delay,
	}
}

func (c *Crawler) newClient(workerIdx int) *hyperliquid.Client {
	if c.proxyMgr != nil {
		client, err := c.proxyMgr.NewClientForWorker(workerIdx)
		if err != nil {
			zap.S().Warnf("[crawler] worker %d: create proxy client error: %v, falling back to direct", workerIdx, err)
			return hyperliquid.NewClient()
		}
		return client
	}
	return hyperliquid.NewClient()
}

// SyncLeaderboard 获取排行榜前5000交易员，保存地址到Trader表，windowPerformances保存到TraderPerformance表
func (c *Crawler) SyncLeaderboard() error {
	zap.S().Info("[crawler] fetching leaderboard...")
	client := c.newClient(0)
	resp, err := client.FetchLeaderboard()
	if err != nil {
		return fmt.Errorf("fetch leaderboard: %w", err)
	}

	rows := resp.LeaderboardRows
	zap.S().Infof("[crawler] got %d leaderboard rows", len(rows))

	// 按30D交易量排序，取前5000
	sort.Slice(rows, func(i, j int) bool {
		return monthVlm(rows[i]) > monthVlm(rows[j])
	})
	if len(rows) > 5000 {
		rows = rows[:5000]
	}
	zap.S().Infof("[crawler] top %d traders by 30D volume", len(rows))

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
				Address:  row.EthAddress,
				Username: row.DisplayName,
			})
		}
		if err := c.db.Select("Address").Clauses(clause.OnConflict{
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
					zap.S().Warnf("[crawler] skip bad windowPerformance for %s: %v", row.EthAddress, err)
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

		zap.S().Infof("[crawler] saved batch %d/%d (%d traders)", i/batchSize+1, (len(rows)+batchSize-1)/batchSize, len(batch))
	}

	zap.S().Info("[crawler] leaderboard sync done")
	return nil
}

// SyncPortfolios 并发获取所有Trader的 accountValueHistory 和 pnlHistory
func (c *Crawler) SyncPortfolios() error {
	var traders []model.Trader
	if err := c.db.Select("address").Find(&traders).Error; err != nil {
		return fmt.Errorf("load traders: %w", err)
	}
	zap.S().Infof("[crawler] syncing portfolios for %d traders with %d workers", len(traders), c.workers)

	addrCh := make(chan string, len(traders))
	for _, t := range traders {
		addrCh <- t.Address
	}
	close(addrCh)

	var (
		wg    sync.WaitGroup
		done  atomic.Int64
		total = int64(len(traders))
	)

	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			client := c.newClient(workerIdx)
			for address := range addrCh {
				if err := c.syncOnePortfolio(client, address); err != nil {
					zap.S().Warnf("[crawler] portfolio error for %s: %v", address, err)
				}
				cur := done.Add(1)
				if cur%100 == 0 || cur == total {
					zap.S().Infof("[crawler] portfolio progress: %d/%d", cur, total)
				}
				if c.delay > 0 {
					time.Sleep(c.delay)
				}
			}
		}(i)
	}

	wg.Wait()
	zap.S().Info("[crawler] portfolio sync done")
	return nil
}

func (c *Crawler) syncOnePortfolio(client *hyperliquid.Client, address string) error {
	entries, err := client.FetchPortfolio(address)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		window, data, err := entry.Parse()
		if err != nil {
			zap.S().Warnf("[crawler] parse portfolio window for %s: %v", address, err)
			continue
		}

		if len(data.PnlHistory) > 0 {
			historyJSON, _ := json.Marshal(data.PnlHistory)
			record := model.TraderPnlHistory{
				Address: address,
				Window:  window,
				History: historyJSON,
			}
			if err := c.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}, {Name: "window"}},
				DoUpdates: clause.AssignmentColumns([]string{"history", "updated_at"}),
			}).Create(&record).Error; err != nil {
				return fmt.Errorf("upsert pnl history (%s): %w", window, err)
			}
		}

		if len(data.AccountValueHistory) > 0 {
			historyJSON, _ := json.Marshal(data.AccountValueHistory)
			record := model.TraderAccountValue{
				Address: address,
				Window:  window,
				History: historyJSON,
			}
			if err := c.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}, {Name: "window"}},
				DoUpdates: clause.AssignmentColumns([]string{"history", "updated_at"}),
			}).Create(&record).Error; err != nil {
				return fmt.Errorf("upsert account value (%s): %w", window, err)
			}
		}
	}

	return nil
}

// monthVlm 提取30D交易量用于排序
func monthVlm(row model.LeaderboardRow) float64 {
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
