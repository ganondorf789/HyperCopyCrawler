package funding

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

var defaultStart = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// Worker 异步 worker 池，并发获取交易员的资金费记录
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

// Run 获取所有交易员的资金费记录
func (w *Worker) Run() error {
	var traders []model.Trader
	if err := w.db.Select("address").Find(&traders).Error; err != nil {
		return err
	}
	zap.S().Infof("[funding] total %d traders to fetch", len(traders))

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
	zap.S().Infof("[funding] all done. %d traders processed, %d funding records saved", done.Load(), saved.Load())
	return nil
}

func (w *Worker) worker(workerIdx int, addrCh <-chan string, done, saved *atomic.Int64, total int64) {
	var client *hyperliquid.Client
	if w.proxyMgr != nil {
		var err error
		client, err = w.proxyMgr.NewClientForWorker(workerIdx)
		if err != nil {
			zap.S().Warnf("[funding] worker %d: create client error: %v", workerIdx, err)
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
			zap.S().Infof("[funding] progress: %d/%d traders, %d funding records saved", cur, total, saved.Load())
		}
	}
}

func (w *Worker) processOne(client *hyperliquid.Client, address string) int {
	var latestFunding model.TraderFunding
	startMs := defaultStart.UnixMilli()
	if err := w.db.Where("address = ?", address).Order("time DESC").First(&latestFunding).Error; err == nil {
		startMs = latestFunding.Time + 1
	}

	endMs := time.Now().UTC().UnixMilli()
	if startMs >= endMs {
		return 0
	}

	entries, abortErr := FetchAllFunding(client, address, startMs, endMs, w.delay)
	if len(entries) == 0 && abortErr == nil {
		return 0
	}

	n := 0
	if len(entries) > 0 {
		n = w.saveFunding(address, entries)
		zap.S().Infof("[funding] %s: fetched %d, saved %d", address[:10], len(entries), n)
	}

	if abortErr != nil {
		zap.S().Warnf("[funding] %s: fetch aborted (%s), skipping trader. %v", address[:10], abortErr.Reason, abortErr)
		w.recordFailure(address, abortErr)
	}

	return n
}

func (w *Worker) recordFailure(address string, e *FetchAbortErr) {
	failure := model.FetchFailure{
		Type:        "funding",
		Reason:      e.Reason,
		Address:     address,
		StartMs:     e.StartMs,
		EndMs:       e.EndMs,
		RecordCount: e.Count,
	}
	if err := w.db.Create(&failure).Error; err != nil {
		zap.S().Warnf("[funding] failed to record fetch failure for %s: %v", address[:10], err)
	}
}

func (w *Worker) saveFunding(address string, entries []model.FundingEntry) int {
	records := make([]model.TraderFunding, 0, len(entries))
	for _, e := range entries {
		records = append(records, model.TraderFunding{
			Address:     address,
			Time:        e.Time,
			Hash:        e.Hash,
			Coin:        e.Delta.Coin,
			Usdc:        e.Delta.Usdc,
			Szi:         e.Delta.Szi,
			FundingRate: e.Delta.FundingRate,
		})
	}

	batch := 500
	saved := 0
	for i := 0; i < len(records); i += batch {
		end := i + batch
		if end > len(records) {
			end = len(records)
		}
		if err := w.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}, {Name: "time"}, {Name: "coin"}},
			DoNothing: true,
		}).Create(records[i:end]).Error; err != nil {
			zap.S().Warnf("[funding] save error for %s batch %d: %v", address[:10], i/batch, err)
			continue
		}
		saved += end - i
	}
	return saved
}
