package fills

import (
	"fmt"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"go.uber.org/zap"
)

// 8层自适应细分策略（从后往前）：月 → 周 → 天 → 小时 → 10分钟 → 2分钟 → 30秒
// 当单次请求返回 >=2000 条时，自动向下一层细分
// 30秒仍然 >=2000 时，保存已获取数据并跳过当前交易员

// ExceedsLimitErr 30秒窗口仍超过2000条限制
type ExceedsLimitErr struct {
	StartMs int64
	EndMs   int64
	Count   int
}

func (e *ExceedsLimitErr) Error() string {
	return fmt.Sprintf("fills exceed 2000 limit in 30s window [%d, %d], got %d", e.StartMs, e.EndMs, e.Count)
}

// FetchAllFills 获取交易员从 startMs 到 endMs 的所有成交记录（自适应8层细分）
// 返回 *ExceedsLimitErr 表示30秒窗口仍超限，调用方应保存已获取数据并跳过该交易员
func FetchAllFills(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	fills, err := client.FetchUserFillsByTime(address, startMs, endMs)
	if err != nil {
		zap.S().Warnf("[fills] probe error for %s: %v", address[:10], err)
		return nil, nil
	}
	if len(fills) == 0 {
		return nil, nil
	}
	if !hyperliquid.IsAtLimit(fills) {
		return fills, nil
	}

	zap.S().Infof("[fills] %s: hit 2000 limit, splitting by month", address[:10])
	return fetchByMonth(client, address, startMs, endMs, delay)
}

// --- Level 1: 按月 ---
func fetchByMonth(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 1, 0)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] month error %s [%s]: %v", address[:10], cur.Format("2006-01"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s month %s hit limit, split by week", address[:10], cur.Format("2006-01"))
			sub, exceedErr := fetchByWeek(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 2: 按周 ---
func fetchByWeek(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 7)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] week error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s week %s hit limit, split by day", address[:10], cur.Format("01-02"))
			sub, exceedErr := fetchByDay(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 3: 按天 ---
func fetchByDay(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 1)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] day error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s day %s hit limit, split by hour", address[:10], cur.Format("01-02"))
			sub, exceedErr := fetchByHour(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 4: 按小时 ---
func fetchByHour(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(time.Hour)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] hour error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s hour %s hit limit, split by 10min", address[:10], cur.Format("15:04"))
			sub, exceedErr := fetchBy10Min(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 5: 按10分钟 ---
func fetchBy10Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(10 * time.Minute)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] 10min error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s 10min %s hit limit, split by 2min", address[:10], cur.Format("15:04"))
			sub, exceedErr := fetchBy2Min(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 6: 按2分钟 ---
func fetchBy2Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(2 * time.Minute)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] 2min error %s [%s]: %v", address[:10], cur.Format("15:04:05"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Infof("[fills] %s 2min %s hit limit, split by 30s", address[:10], cur.Format("15:04:05"))
			sub, exceedErr := fetchBy30Sec(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if exceedErr != nil {
				return all, exceedErr
			}
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 7: 按30秒（最细粒度） ---
func fetchBy30Sec(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.Fill, *ExceedsLimitErr) {
	var all []model.Fill
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(30 * time.Second)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		fills, err := client.FetchUserFillsByTime(address, cMs, nMs)
		if err != nil {
			zap.S().Warnf("[fills] 30s error %s [%s]: %v", address[:10], cur.Format("15:04:05"), err)
			cur = next
			sleep(delay)
			continue
		}

		all = append(all, fills...)

		if hyperliquid.IsAtLimit(fills) {
			zap.S().Warnf("[fills] %s 30s %s still at limit (%d), cannot split further — skipping trader",
				address[:10], cur.Format("15:04:05"), len(fills))
			return all, &ExceedsLimitErr{
				StartMs: cMs,
				EndMs:   nMs,
				Count:   len(fills),
			}
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

func sleep(d time.Duration) {
	if d > 0 {
		time.Sleep(d)
	}
}
