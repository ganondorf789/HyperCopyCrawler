package snapshot

import (
	"log"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/proxy"
	"gorm.io/gorm"
)

type Syncer struct {
	db       *gorm.DB
	proxyMgr *proxy.Manager
	workers  int
}

func NewSyncer(db *gorm.DB, proxyMgr *proxy.Manager, workers int) *Syncer {
	return &Syncer{
		db:       db,
		proxyMgr: proxyMgr,
		workers:  workers,
	}
}

func (s *Syncer) Run() {
	for round := 1; ; round++ {
		var traders []model.Trader
		if err := s.db.Select("address").Find(&traders).Error; err != nil {
			log.Printf("[snapshot] load traders error: %v, retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		total := int64(len(traders))
		log.Printf("[snapshot] round %d: %d traders to sync", round, total)

		addrCh := make(chan string, len(traders))
		for _, t := range traders {
			addrCh <- t.Address
		}
		close(addrCh)

		var (
			wg   sync.WaitGroup
			done atomic.Int64
			errs atomic.Int64
		)

		for i := 0; i < s.workers; i++ {
			wg.Add(1)
			go func(workerIdx int) {
				defer wg.Done()
				s.worker(workerIdx, addrCh, &done, &errs, total)
			}(i)
		}

		wg.Wait()
		log.Printf("[snapshot] round %d done: %d/%d succeeded, %d errors",
			round, done.Load()-errs.Load(), total, errs.Load())
	}
}

func (s *Syncer) worker(workerIdx int, addrCh <-chan string, done, errs *atomic.Int64, total int64) {
	client, err := s.proxyMgr.NewClientForWorker(workerIdx)
	if err != nil {
		log.Printf("[snapshot] worker %d: create client error: %v", workerIdx, err)
		return
	}

	for address := range addrCh {
		if err := s.processOne(client, address); err != nil {
			log.Printf("[snapshot] %s error: %v", address[:10], err)
			errs.Add(1)
		}
		cur := done.Add(1)
		if cur%100 == 0 || cur == total {
			log.Printf("[snapshot] progress: %d/%d", cur, total)
		}
	}
}

func (s *Syncer) processOne(client *hyperliquid.Client, address string) error {
	chState, err := client.FetchClearinghouseState(address)
	if err != nil {
		return err
	}

	spotState, err := client.FetchSpotClearinghouseState(address)
	if err != nil {
		return err
	}

	if err := s.upsertPositions(address, chState.AssetPositions); err != nil {
		return err
	}

	return s.updateTraderSnap(address, chState, spotState)
}

func (s *Syncer) upsertPositions(address string, positions []model.AssetPosition) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("address = ?", address).Delete(&model.TraderPosition{}).Error; err != nil {
			return err
		}
		if len(positions) == 0 {
			return nil
		}

		records := make([]model.TraderPosition, 0, len(positions))
		for _, ap := range positions {
			p := ap.Position
			liqPx := ""
			if p.LiquidationPx != nil {
				liqPx = *p.LiquidationPx
			}
			records = append(records, model.TraderPosition{
				Address:               address,
				Coin:                  p.Coin,
				Szi:                   p.Szi,
				LeverageType:          p.Leverage.Type,
				Leverage:              p.Leverage.Value,
				EntryPx:               p.EntryPx,
				PositionValue:         p.PositionValue,
				UnrealizedPnl:         p.UnrealizedPnl,
				ReturnOnEquity:        p.ReturnOnEquity,
				LiquidationPx:         liqPx,
				MarginUsed:            p.MarginUsed,
				MaxLeverage:           p.MaxLeverage,
				CumFundingAllTime:     p.CumFunding.AllTime,
				CumFundingSinceOpen:   p.CumFunding.SinceOpen,
				CumFundingSinceChange: p.CumFunding.SinceChange,
			})
		}
		return tx.Create(&records).Error
	})
}

func (s *Syncer) updateTraderSnap(address string, ch *model.ClearinghouseState, spot *model.SpotClearinghouseState) error {
	zero := new(big.Float)
	longCount, shortCount := 0, 0
	longValue := new(big.Float)
	shortValue := new(big.Float)
	longPnl := new(big.Float)
	shortPnl := new(big.Float)
	totalUnrealizedPnl := new(big.Float)

	for _, ap := range ch.AssetPositions {
		p := ap.Position

		szi, _, _ := new(big.Float).Parse(p.Szi, 10)
		if szi == nil {
			continue
		}

		pv, _, _ := new(big.Float).Parse(p.PositionValue, 10)
		if pv == nil {
			pv = new(big.Float)
		}

		upnl, _, _ := new(big.Float).Parse(p.UnrealizedPnl, 10)
		if upnl == nil {
			upnl = new(big.Float)
		}
		totalUnrealizedPnl.Add(totalUnrealizedPnl, upnl)

		if szi.Cmp(zero) > 0 {
			longCount++
			longValue.Add(longValue, pv)
			longPnl.Add(longPnl, upnl)
		} else if szi.Cmp(zero) < 0 {
			shortCount++
			shortValue.Add(shortValue, pv)
			shortPnl.Add(shortPnl, upnl)
		}
	}

	spotValue := new(big.Float)
	for _, b := range spot.Balances {
		ntl, _, _ := new(big.Float).Parse(b.EntryNtl, 10)
		if ntl != nil {
			spotValue.Add(spotValue, ntl)
		}
	}

	accountValue, _, _ := new(big.Float).Parse(ch.MarginSummary.AccountValue, 10)
	if accountValue == nil {
		accountValue = new(big.Float)
	}
	totalNtlPos, _, _ := new(big.Float).Parse(ch.MarginSummary.TotalNtlPos, 10)
	if totalNtlPos == nil {
		totalNtlPos = new(big.Float)
	}
	totalMarginUsed, _, _ := new(big.Float).Parse(ch.MarginSummary.TotalMarginUsed, 10)
	if totalMarginUsed == nil {
		totalMarginUsed = new(big.Float)
	}

	effLeverage := new(big.Float)
	marginUsageRate := new(big.Float)
	if accountValue.Cmp(zero) != 0 {
		effLeverage.Quo(totalNtlPos, accountValue)
		marginUsageRate.Quo(totalMarginUsed, accountValue)
	}

	totalValue := new(big.Float).Add(accountValue, spotValue)

	updates := map[string]interface{}{
		"snap_eff_leverage":          effLeverage.Text('f', 10),
		"snap_long_position_count":   longCount,
		"snap_long_position_value":   longValue.Text('f', 10),
		"snap_margin_usage_rate":     marginUsageRate.Text('f', 10),
		"snap_perp_value":            ch.CrossMarginSummary.TotalNtlPos,
		"snap_position_count":        longCount + shortCount,
		"snap_position_value":        ch.MarginSummary.TotalNtlPos,
		"snap_short_position_count":  shortCount,
		"snap_short_position_value":  shortValue.Text('f', 10),
		"snap_spot_value":            spotValue.Text('f', 10),
		"snap_total_margin_used":     ch.MarginSummary.TotalMarginUsed,
		"snap_total_value":           totalValue.Text('f', 10),
		"snap_unrealized_pnl":        totalUnrealizedPnl.Text('f', 10),
		"long_pnl":                   longPnl.Text('f', 10),
		"short_pnl":                  shortPnl.Text('f', 10),
	}

	return s.db.Model(&model.Trader{}).Where("address = ?", address).Updates(updates).Error
}
