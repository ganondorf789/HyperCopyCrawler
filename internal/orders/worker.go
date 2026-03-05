package orders

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

// Worker 异步 worker 池，并发获取交易员的历史委托记录
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

// Run 获取所有交易员的历史委托记录
func (w *Worker) Run() error {
	var traders []model.Trader
	if err := w.db.Select("address").Find(&traders).Error; err != nil {
		return err
	}
	zap.S().Infof("[orders] total %d traders to fetch", len(traders))

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
	zap.S().Infof("[orders] all done. %d traders processed, %d orders saved", done.Load(), saved.Load())
	return nil
}

func (w *Worker) worker(workerIdx int, addrCh <-chan string, done, saved *atomic.Int64, total int64) {
	var client *hyperliquid.Client
	if w.proxyMgr != nil {
		var err error
		client, err = w.proxyMgr.NewClientForWorker(workerIdx)
		if err != nil {
			zap.S().Warnf("[orders] worker %d: create client error: %v", workerIdx, err)
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
			zap.S().Infof("[orders] progress: %d/%d traders, %d orders saved", cur, total, saved.Load())
		}
	}
}

func (w *Worker) processOne(client *hyperliquid.Client, address string) int {
	var latestOrder model.TraderOrder
	startMs := defaultStart.UnixMilli()
	if err := w.db.Where("address = ?", address).Order("timestamp DESC").First(&latestOrder).Error; err == nil {
		startMs = latestOrder.Timestamp + 1
	}

	endMs := time.Now().UTC().UnixMilli()
	if startMs >= endMs {
		return 0
	}

	entries, exceedErr := FetchAllOrders(client, address, startMs, endMs, w.delay)
	if len(entries) == 0 && exceedErr == nil {
		return 0
	}

	n := 0
	if len(entries) > 0 {
		n = w.saveOrders(address, entries)
	}

	if exceedErr != nil {
		zap.S().Warnf("[orders] %s: exceeded limit at 30s granularity, skipping trader. %v", address[:10], exceedErr)
		w.recordFailure(address, exceedErr)
	}

	return n
}

func (w *Worker) recordFailure(address string, e *ExceedsLimitErr) {
	failure := model.FetchFailure{
		Type:        "orders",
		Address:     address,
		StartMs:     e.StartMs,
		EndMs:       e.EndMs,
		RecordCount: e.Count,
	}
	if err := w.db.Create(&failure).Error; err != nil {
		zap.S().Warnf("[orders] failed to record fetch failure for %s: %v", address[:10], err)
	}
}

func (w *Worker) saveOrders(address string, entries []model.OrderEntry) int {
	records := make([]model.TraderOrder, 0, len(entries))
	for _, e := range entries {
		children := "[]"
		if len(e.Children) > 0 {
			children = string(e.Children)
		}
		records = append(records, model.TraderOrder{
			Address:          address,
			Coin:             e.Coin,
			Side:             e.Side,
			LimitPx:          e.LimitPx,
			Sz:               e.Sz,
			Oid:              e.Oid,
			Timestamp:        e.Timestamp,
			TriggerCondition: e.TriggerCondition,
			IsTrigger:        e.IsTrigger,
			TriggerPx:        e.TriggerPx,
			Children:         children,
			IsPositionTpsl:   e.IsPositionTpsl,
			ReduceOnly:       e.ReduceOnly,
			OrderType:        e.OrderType,
			OrigSz:           e.OrigSz,
			Tif:              e.Tif,
			Cloid:            e.Cloid,
			Status:           e.Status,
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
			Columns:   []clause.Column{{Name: "address"}, {Name: "oid"}},
			DoNothing: true,
		}).Create(records[i:end]).Error; err != nil {
			zap.S().Warnf("[orders] save error for %s batch %d: %v", address[:10], i/batch, err)
			continue
		}
		saved += end - i
	}
	return saved
}
