package fills

import (
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

// 默认时间范围：2024-01-01 ~ 现在
var defaultStart = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// Worker 异步 worker 池，并发获取交易员的成交记录
type Worker struct {
	db       *gorm.DB
	proxyMgr *proxy.Manager
	workers  int
	delay    time.Duration
}

func NewWorker(db *gorm.DB, proxyMgr *proxy.Manager, workers int, delay time.Duration) *Worker {
	return &Worker{
		db:       db,
		proxyMgr: proxyMgr,
		workers:  workers,
		delay:    delay,
	}
}

// Run 获取所有交易员的成交记录
func (w *Worker) Run() error {
	var traders []model.Trader
	if err := w.db.Select("address").Find(&traders).Error; err != nil {
		return err
	}
	zap.S().Infof("[fills] total %d traders to fetch", len(traders))

	// 地址 channel
	addrCh := make(chan string, len(traders))
	for _, t := range traders {
		addrCh <- t.Address
	}
	close(addrCh)

	var (
		wg    sync.WaitGroup
		done  atomic.Int64
		saved atomic.Int64
		total = int64(len(traders))
	)

	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			w.worker(workerIdx, addrCh, &done, &saved, total)
		}(i)
	}

	wg.Wait()
	zap.S().Infof("[fills] all done. %d traders processed, %d fills saved", done.Load(), saved.Load())
	return nil
}

func (w *Worker) worker(workerIdx int, addrCh <-chan string, done, saved *atomic.Int64, total int64) {
	var client *hyperliquid.Client
	if w.proxyMgr != nil {
		var err error
		client, err = w.proxyMgr.NewClientForWorker(workerIdx)
		if err != nil {
			zap.S().Warnf("[fills] worker %d: create client error: %v", workerIdx, err)
			return
		}
	} else {
		client = hyperliquid.NewClient()
	}

	for address := range addrCh {
		n := w.processOne(client, address)
		saved.Add(int64(n))
		cur := done.Add(1)
		if cur%50 == 0 || cur == total {
			zap.S().Infof("[fills] progress: %d/%d traders, %d fills saved", cur, total, saved.Load())
		}
	}
}

func (w *Worker) processOne(client *hyperliquid.Client, address string) int {
	var latestFill model.TraderFill
	startMs := defaultStart.UnixMilli()
	if err := w.db.Where("address = ?", address).Order("time DESC").First(&latestFill).Error; err == nil {
		startMs = latestFill.Time + 1
	}

	endMs := time.Now().UTC().UnixMilli()
	if startMs >= endMs {
		return 0
	}

	fills, exceedErr := FetchAllFills(client, address, startMs, endMs, w.delay)
	if len(fills) == 0 && exceedErr == nil {
		return 0
	}

	n := 0
	if len(fills) > 0 {
		n = w.saveFills(address, fills)
	}

	if exceedErr != nil {
		zap.S().Warnf("[fills] %s: exceeded limit at 30s granularity, skipping trader. %v", address[:10], exceedErr)
		w.recordFailure(address, exceedErr)
		return n
	}

	if err := BuildCompletedTrades(w.db, address); err != nil {
		zap.S().Warnf("[fills] rebuild trades error for %s: %v", address[:10], err)
	}

	return n
}

func (w *Worker) recordFailure(address string, e *ExceedsLimitErr) {
	failure := model.FillFetchFailure{
		Address:   address,
		StartMs:   e.StartMs,
		EndMs:     e.EndMs,
		FillCount: e.Count,
	}
	if err := w.db.Create(&failure).Error; err != nil {
		zap.S().Warnf("[fills] failed to record fetch failure for %s: %v", address[:10], err)
	}
}

func (w *Worker) saveFills(address string, fills []model.Fill) int {
	records := make([]model.TraderFill, 0, len(fills))
	for _, f := range fills {
		records = append(records, model.TraderFill{
			Address:       address,
			Coin:          f.Coin,
			Px:            f.Px,
			Sz:            f.Sz,
			Side:          f.Side,
			Time:          f.Time,
			StartPosition: f.StartPosition,
			Dir:           f.Dir,
			ClosedPnl:     f.ClosedPnl,
			Hash:          f.Hash,
			Oid:           f.Oid,
			Crossed:       f.Crossed,
			Fee:           f.Fee,
			Tid:           f.Tid,
			Cloid:         f.Cloid,
			FeeToken:      f.FeeToken,
		})
	}

	// 分批写入，每批 500 条
	batch := 500
	saved := 0
	for i := 0; i < len(records); i += batch {
		end := i + batch
		if end > len(records) {
			end = len(records)
		}
		if err := w.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tid"}},
			DoNothing: true,
		}).Create(records[i:end]).Error; err != nil {
			zap.S().Warnf("[fills] save error for %s batch %d: %v", address[:10], i/batch, err)
			continue
		}
		saved += end - i
	}
	return saved
}
