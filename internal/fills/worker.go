package fills

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/proxy"
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
	log.Printf("[fills] total %d traders to fetch", len(traders))

	// 地址 channel
	addrCh := make(chan string, len(traders))
	for _, t := range traders {
		addrCh <- t.Address
	}
	close(addrCh)

	var (
		wg      sync.WaitGroup
		done    atomic.Int64
		saved   atomic.Int64
		total   = int64(len(traders))
	)

	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			w.worker(workerIdx, addrCh, &done, &saved, total)
		}(i)
	}

	wg.Wait()
	log.Printf("[fills] all done. %d traders processed, %d fills saved", done.Load(), saved.Load())
	return nil
}

func (w *Worker) worker(workerIdx int, addrCh <-chan string, done, saved *atomic.Int64, total int64) {
	client, err := w.proxyMgr.NewClientForWorker(workerIdx)
	if err != nil {
		log.Printf("[fills] worker %d: create client error: %v", workerIdx, err)
		return
	}

	for address := range addrCh {
		n := w.processOne(client, address)
		saved.Add(int64(n))
		cur := done.Add(1)
		if cur%50 == 0 || cur == total {
			log.Printf("[fills] progress: %d/%d traders, %d fills saved", cur, total, saved.Load())
		}
	}
}

func (w *Worker) processOne(client *hyperliquid.Client, address string) int {
	// 查询该交易员已有的最新 fill 时间
	var latestFill model.TraderFill
	startMs := defaultStart.UnixMilli()
	if err := w.db.Where("address = ?", address).Order("time DESC").First(&latestFill).Error; err == nil {
		// 从最新记录之后开始（+1ms 避免重复）
		startMs = latestFill.Time + 1
	}

	endMs := time.Now().UTC().UnixMilli()
	if startMs >= endMs {
		return 0
	}

	fills := FetchAllFills(client, address, startMs, endMs, w.delay)
	if len(fills) == 0 {
		return 0
	}

	// 批量写入数据库
	return w.saveFills(address, fills)
}

func (w *Worker) saveFills(address string, fills []hyperliquid.Fill) int {
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
			log.Printf("[fills] save error for %s batch %d: %v", address[:10], i/batch, err)
			continue
		}
		saved += end - i
	}
	return saved
}
