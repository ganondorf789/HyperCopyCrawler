package snapshot

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/utility"
	"github.com/lib/pq"
	"gorm.io/gorm/clause"
)

var allWindows = []string{"day", "week", "month", "allTime"}

func (s *Syncer) updateStatistics(address string) error {
	var trader model.Trader
	if err := s.db.Where("address = ?", address).First(&trader).Error; err != nil {
		return fmt.Errorf("load trader: %w", err)
	}

	var trades []model.CompletedTrade
	s.db.Where("address = ?", address).Find(&trades)

	avMap := make(map[string][]byte)
	var accountValues []model.TraderAccountValue
	s.db.Where("address = ?", address).Find(&accountValues)
	for _, av := range accountValues {
		avMap[av.Window] = []byte(av.History)
	}

	for _, window := range allWindows {
		stat := buildStat(address, window, &trader, trades, avMap[window])
		if err := s.upsertStat(&stat); err != nil {
			return fmt.Errorf("upsert %s/%s: %w", utility.Abbr(address), window, err)
		}
	}

	labels := computeLabels(&trader, trades, avMap["allTime"])
	if labels == nil {
		labels = []string{}
	}
	if err := s.db.Model(&trader).Update("labels", pq.StringArray(labels)).Error; err != nil {
		return fmt.Errorf("update labels %s: %w", utility.Abbr(address), err)
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

func (s *Syncer) upsertStat(stat *model.TraderStatistic) error {
	return s.db.Clauses(clause.OnConflict{
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
