package statistic

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/utility"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var allWindows = []string{"day", "week", "month", "allTime"}

type Calculator struct {
	db      *gorm.DB
	workers int
}

func NewCalculator(db *gorm.DB, workers int) *Calculator {
	return &Calculator{db: db, workers: workers}
}

func (c *Calculator) Run() {
	var traders []model.Trader
	if err := c.db.Select("address").Find(&traders).Error; err != nil {
		zap.S().Fatalf("[statistic] load traders: %v", err)
	}

	total := int64(len(traders))
	zap.S().Infof("[statistic] %d traders to process", total)

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

	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for address := range addrCh {
				if err := c.processTrader(address); err != nil {
					zap.S().Warnf("[statistic] %s error: %v", utility.Abbr(address), err)
					errs.Add(1)
				}
				cur := done.Add(1)
				if cur%100 == 0 || cur == total {
					zap.S().Infof("[statistic] progress: %d/%d", cur, total)
				}
			}
		}()
	}

	wg.Wait()
	zap.S().Infof("[statistic] done: %d/%d succeeded, %d errors",
		done.Load()-errs.Load(), total, errs.Load())
}

// ── per-trader pipeline ─────────────────────────────────────────────

func (c *Calculator) processTrader(address string) error {
	var trader model.Trader
	if err := c.db.Where("address = ?", address).First(&trader).Error; err != nil {
		return fmt.Errorf("load trader: %w", err)
	}

	var trades []model.CompletedTrade
	c.db.Where("address = ?", address).Find(&trades)

	avMap := make(map[string][]byte)
	var accountValues []model.TraderAccountValue
	c.db.Where("address = ?", address).Find(&accountValues)
	for _, av := range accountValues {
		avMap[av.Window] = []byte(av.History)
	}

	for _, window := range allWindows {
		stat := buildStat(address, window, &trader, trades, avMap[window])
		if err := c.upsert(&stat); err != nil {
			return fmt.Errorf("upsert %s/%s: %w", utility.Abbr(address), window, err)
		}
	}
	return nil
}

// ── trade stats (window-dependent) ──────────────────────────────────

type tradeStats struct {
	profitCount int
	totalPnl    float64
	winRate     float64

	longCount       int
	longProfitCount int
	longPnl         float64
	longWinRate     float64

	shortCount       int
	shortProfitCount int
	shortPnl         float64
	shortWinRate     float64
}

func calcTradeStats(trades []model.CompletedTrade, cutoff int64) tradeStats {
	var ts tradeStats
	total := 0

	for _, t := range trades {
		if t.EndTime < cutoff {
			continue
		}
		total++

		if t.Pnl > 0 {
			ts.profitCount++
		}
		ts.totalPnl += t.Pnl

		switch t.Direction {
		case "long":
			ts.longCount++
			ts.longPnl += t.Pnl
			if t.Pnl > 0 {
				ts.longProfitCount++
			}
		case "short":
			ts.shortCount++
			ts.shortPnl += t.Pnl
			if t.Pnl > 0 {
				ts.shortProfitCount++
			}
		}
	}

	if total > 0 {
		ts.winRate = float64(ts.profitCount) / float64(total)
	}
	if ts.longCount > 0 {
		ts.longWinRate = float64(ts.longProfitCount) / float64(ts.longCount)
	}
	if ts.shortCount > 0 {
		ts.shortWinRate = float64(ts.shortProfitCount) / float64(ts.shortCount)
	}
	return ts
}

// ── sharpe & drawdown (from account value history) ──────────────────

func parseTimeSeries(data []byte) [][2]float64 {
	if len(data) == 0 {
		return nil
	}
	var raw [][]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := make([][2]float64, 0, len(raw))
	for _, pair := range raw {
		if len(pair) < 2 {
			continue
		}
		out = append(out, [2]float64{utility.AnyFloat(pair[0]), utility.AnyFloat(pair[1])})
	}
	return out
}

func calcSharpe(history []byte) float64 {
	series := parseTimeSeries(history)
	if len(series) < 3 {
		return 0
	}

	returns := make([]float64, 0, len(series)-1)
	for i := 1; i < len(series); i++ {
		prev := series[i-1][1]
		if prev == 0 {
			continue
		}
		returns = append(returns, (series[i][1]-prev)/math.Abs(prev))
	}
	if len(returns) < 2 {
		return 0
	}

	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	sumSq := 0.0
	for _, r := range returns {
		d := r - mean
		sumSq += d * d
	}
	std := math.Sqrt(sumSq / float64(len(returns)-1))
	if std == 0 {
		return 0
	}

	return (mean / std) * math.Sqrt(365)
}

func calcMaxDrawdown(history []byte) float64 {
	series := parseTimeSeries(history)
	if len(series) < 2 {
		return 0
	}

	peak := series[0][1]
	maxDD := 0.0
	for _, pt := range series {
		v := pt[1]
		if v > peak {
			peak = v
		}
		if peak > 0 {
			dd := (peak - v) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// ── assemble stat row ───────────────────────────────────────────────

func buildStat(
	address, window string,
	trader *model.Trader,
	allTrades []model.CompletedTrade,
	accountValueHistory []byte,
) model.TraderStatistic {
	cutoff := utility.WindowCutoff(window)
	ts := calcTradeStats(allTrades, cutoff)
	sharpe := calcSharpe(accountValueHistory)
	drawdown := calcMaxDrawdown(accountValueHistory)

	return model.TraderStatistic{
		Address:            address,
		Window:             window,
		Sharpe:             utility.FmtFloat(sharpe),
		Drawdown:           utility.FmtFloat(drawdown),
		PositionCount:      strconv.Itoa(trader.SnapPositionCount),
		TotalValue:         utility.OrZero(trader.SnapTotalValue),
		PerpValue:          utility.OrZero(trader.SnapPerpValue),
		PositionValue:      utility.OrZero(trader.SnapPositionValue),
		LongPositionValue:  utility.OrZero(trader.SnapLongPositionValue),
		ShortPositionValue: utility.OrZero(trader.SnapShortPositionValue),
		MarginUsage:        utility.OrZero(trader.SnapMarginUsageRate),
		UsedMargin:         utility.OrZero(trader.SnapTotalMarginUsed),
		ProfitCount:        strconv.Itoa(ts.profitCount),
		WinRate:            utility.FmtFloat(ts.winRate),
		TotalPnl:           utility.FmtFloat(ts.totalPnl),
		LongCount:          strconv.Itoa(ts.longCount),
		LongRealizedPnl:    utility.FmtFloat(ts.longPnl),
		LongWinRate:        utility.FmtFloat(ts.longWinRate),
		ShortCount:         strconv.Itoa(ts.shortCount),
		ShortRealizedPnl:   utility.FmtFloat(ts.shortPnl),
		ShortWinRate:       utility.FmtFloat(ts.shortWinRate),
		UnrealizedPnl:      utility.OrZero(trader.SnapUnrealizedPnl),
		AvgLeverage:        utility.OrZero(trader.SnapEffLeverage),
	}
}

// ── upsert ──────────────────────────────────────────────────────────

func (c *Calculator) upsert(stat *model.TraderStatistic) error {
	return c.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "address"}, {Name: "window"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"sharpe", "drawdown", "position_count", "total_value", "perp_value",
			"position_value", "long_position_value", "short_position_value",
			"margin_usage", "used_margin", "profit_count", "win_rate",
			"total_pnl", "long_count", "long_realized_pnl", "long_win_rate",
			"short_count", "short_realized_pnl", "short_win_rate",
			"unrealized_pnl", "avg_leverage", "updated_at",
		}),
	}).Create(stat).Error
}
