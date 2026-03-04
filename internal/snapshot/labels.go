package snapshot

import (
	"strconv"
	"time"

	"github.com/hypercopy/crawler/internal/consts"
	"github.com/hypercopy/crawler/internal/model"
)

const (
	msPerHour = 3_600_000
	msPerDay  = 86_400_000
	msPerWeek = 604_800_000
)

type labelMetrics struct {
	totalValue      float64
	avgHoldingMs    float64
	activeDays90    int
	tradeCount      int
	tradeCount90    int
	tradeCount30    int
	totalROI        float64
	profitFactor    float64
	expectedValue   float64
	rolling30Return float64
	longRatio       float64
	winRate         float64
	riskRewardRatio float64
	maxDrawdown     float64
	maxDrawdown90   float64
	sharpe          float64
	avgProfitPerWin float64
	hasSeries90     bool
}

func computeLabels(trader *model.Trader, trades []model.CompletedTrade, accountValueHistory []byte) []string {
	m := calcLabelMetrics(trader, trades, accountValueHistory)
	if m.tradeCount == 0 {
		return nil
	}

	var labels []string

	if l := accountSizeLabel(m); l != "" {
		labels = append(labels, l)
	}
	if l := tradingRhythmLabel(m); l != "" {
		labels = append(labels, l)
	}
	if l := profitStatusLabel(m); l != "" {
		labels = append(labels, l)
	}
	if l := directionLabel(m); l != "" {
		labels = append(labels, l)
	}
	labels = append(labels, tradingStyleLabels(m)...)
	if l := profitScaleLabel(m); l != "" {
		labels = append(labels, l)
	}

	return labels
}

func calcLabelMetrics(trader *model.Trader, trades []model.CompletedTrade, accountValueHistory []byte) labelMetrics {
	var m labelMetrics

	tv, _ := strconv.ParseFloat(trader.SnapTotalValue, 64)
	m.totalValue = tv

	m.sharpe = calcSharpe(accountValueHistory)
	m.maxDrawdown = calcMaxDrawdown(accountValueHistory)

	series := parseTimeSeries(accountValueHistory)

	if len(series) >= 2 {
		first := series[0][1]
		last := series[len(series)-1][1]
		if first > 0 {
			m.totalROI = (last - first) / first
		}
	}

	now := time.Now()

	cutoff30Ts := float64(now.Add(-30 * 24 * time.Hour).UnixMilli())
	var val30DaysAgo float64
	for _, pt := range series {
		if pt[0] >= cutoff30Ts {
			val30DaysAgo = pt[1]
			break
		}
	}
	if val30DaysAgo > 0 && len(series) > 0 {
		m.rolling30Return = (series[len(series)-1][1] - val30DaysAgo) / val30DaysAgo
	}

	cutoff90Ts := float64(now.Add(-90 * 24 * time.Hour).UnixMilli())
	var series90 [][2]float64
	for _, pt := range series {
		if pt[0] >= cutoff90Ts {
			series90 = append(series90, pt)
		}
	}
	m.hasSeries90 = len(series90) >= 2
	if m.hasSeries90 {
		peak := series90[0][1]
		for _, pt := range series90 {
			if pt[1] > peak {
				peak = pt[1]
			}
			if peak > 0 {
				dd := (peak - pt[1]) / peak
				if dd > m.maxDrawdown90 {
					m.maxDrawdown90 = dd
				}
			}
		}
	}

	// ── trade-level metrics ─────────────────────────────────────────
	cutoff90Ms := now.Add(-90 * 24 * time.Hour).UnixMilli()
	cutoff30Ms := now.Add(-30 * 24 * time.Hour).UnixMilli()

	var (
		totalHoldingMs float64
		totalProfit    float64
		totalLoss      float64
		winCount       int
		lossCount      int
		longCount      int
		activeDaySet   = make(map[string]struct{})
	)

	for _, t := range trades {
		m.tradeCount++
		totalHoldingMs += float64(t.EndTime - t.StartTime)

		if t.Pnl > 0 {
			winCount++
			totalProfit += t.Pnl
		} else if t.Pnl < 0 {
			lossCount++
			totalLoss += -t.Pnl
		}

		if t.Direction == "long" {
			longCount++
		}

		if t.EndTime >= cutoff90Ms {
			m.tradeCount90++
			day := time.UnixMilli(t.EndTime).Format("2006-01-02")
			activeDaySet[day] = struct{}{}
		}
		if t.EndTime >= cutoff30Ms {
			m.tradeCount30++
		}
	}

	m.activeDays90 = len(activeDaySet)

	if m.tradeCount > 0 {
		m.avgHoldingMs = totalHoldingMs / float64(m.tradeCount)
		m.winRate = float64(winCount) / float64(m.tradeCount)
		m.longRatio = float64(longCount) / float64(m.tradeCount)
		m.expectedValue = (totalProfit - totalLoss) / float64(m.tradeCount)
	}

	if totalLoss > 0 {
		m.profitFactor = totalProfit / totalLoss
	} else if totalProfit > 0 {
		m.profitFactor = 999
	}

	if winCount > 0 {
		m.avgProfitPerWin = totalProfit / float64(winCount)
	}

	avgLoss := 0.0
	if lossCount > 0 {
		avgLoss = totalLoss / float64(lossCount)
	}
	if avgLoss > 0 && winCount > 0 {
		m.riskRewardRatio = m.avgProfitPerWin / avgLoss
	} else if winCount > 0 {
		m.riskRewardRatio = 999
	}

	return m
}

// ── label assignment ────────────────────────────────────────────────

func accountSizeLabel(m labelMetrics) string {
	switch {
	case m.totalValue > 500_000:
		return consts.AccountTotalValueWhale
	case m.totalValue >= 100_000:
		return consts.AccountTotalValueMedium
	case m.totalValue > 0:
		return consts.AccountTotalValueSmall
	default:
		return ""
	}
}

func tradingRhythmLabel(m labelMetrics) string {
	switch {
	case m.avgHoldingMs > msPerWeek:
		return consts.TradingRhythmLongTerm
	case m.avgHoldingMs > msPerDay:
		return consts.TradingRhythmSwing
	case m.avgHoldingMs >= msPerHour:
		return consts.TradingRhythmShortTerm
	default:
		return consts.TradingRhythmScalping
	}
}

func profitStatusLabel(m labelMetrics) string {
	roiPct := m.totalROI * 100

	if roiPct >= -5 && roiPct <= 5 {
		return consts.ProfitStatusBreakeven
	}

	if m.activeDays90 >= 30 && m.tradeCount90 >= 30 && roiPct > 5 {
		if m.profitFactor >= 1.3 && m.expectedValue > 0 && m.rolling30Return > 0 {
			return consts.ProfitStatusConsistent
		}
		if m.profitFactor < 1.3 || m.rolling30Return <= 0 {
			return consts.ProfitStatusVolatile
		}
	}

	return ""
}

func directionLabel(m labelMetrics) string {
	longPct := m.longRatio * 100
	switch {
	case longPct < 30:
		return consts.DirectionPreferenceBearish
	case longPct > 70:
		return consts.DirectionPreferenceBullish
	default:
		return consts.DirectionPreferenceNeutral
	}
}

func tradingStyleLabels(m labelMetrics) []string {
	var labels []string
	avgHoldDays := m.avgHoldingMs / msPerDay

	if avgHoldDays <= 1 && m.winRate > 0.6 &&
		m.riskRewardRatio >= 1.2 && m.profitFactor >= 1.2 && m.tradeCount30 >= 20 {
		labels = append(labels, consts.TradingStyleHFSteady)
	}

	if avgHoldDays <= 1 && m.winRate < 0.5 &&
		m.riskRewardRatio >= 5 && m.profitFactor >= 1.5 && m.tradeCount30 >= 20 {
		labels = append(labels, consts.TradingStyleHFAggressive)
	}

	if avgHoldDays > 7 && m.winRate > 0.6 &&
		m.riskRewardRatio >= 1.2 && m.profitFactor >= 1.2 && m.tradeCount30 <= 10 {
		labels = append(labels, consts.TradingStyleLFSteady)
	}

	if m.winRate >= 0.6 && m.riskRewardRatio >= 1.5 &&
		m.maxDrawdown <= 0.25 && m.profitFactor >= 1.5 && m.sharpe >= 1.0 {
		labels = append(labels, consts.TradingStyleStableProfit)
	}

	if m.riskRewardRatio >= 3 && m.avgProfitPerWin >= 10_000 && m.maxDrawdown >= 0.3 {
		labels = append(labels, consts.TradingStyleHighRiskReward)
	}

	if m.winRate < 0.5 && m.riskRewardRatio >= 5 &&
		m.profitFactor >= 1.5 && m.tradeCount30 <= 20 {
		labels = append(labels, consts.TradingStyleAsymmetric)
	}

	if m.hasSeries90 && m.maxDrawdown90 <= 0.2 {
		labels = append(labels, consts.TradingStyleLowDrawdown)
	}

	if m.sharpe < 0.8 && m.profitFactor >= 1.0 {
		labels = append(labels, consts.TradingStyleVolatility)
	}

	return labels
}

func profitScaleLabel(m labelMetrics) string {
	if m.avgProfitPerWin == 0 {
		return ""
	}
	switch {
	case m.avgProfitPerWin > 50_000:
		return consts.ProfitScaleLarge
	case m.avgProfitPerWin >= 3_000:
		return consts.ProfitScaleMedium
	default:
		return consts.ProfitScaleSmall
	}
}
