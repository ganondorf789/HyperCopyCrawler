package orders

import (
	"errors"
	"fmt"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"go.uber.org/zap"
)

// 7层自适应细分策略（从粗到细）：月 → 周 → 天 → 小时 → 10分钟 → 2分钟 → 30秒
// 当单次请求返回 >=2000 条时，自动向下一层细分
// 30秒仍然 >=2000 时，保存已获取数据并跳过当前交易员
// 429限频时重试3次，仍失败则保存已获取数据并跳过当前交易员

// FetchAbortErr 获取中断错误（30秒窗口超限 或 429限频重试耗尽）
type FetchAbortErr struct {
	Reason  string // "exceeds_limit" | "rate_limited"
	StartMs int64
	EndMs   int64
	Count   int
}

func (e *FetchAbortErr) Error() string {
	return fmt.Sprintf("orders fetch aborted (%s) in window [%d, %d], count %d", e.Reason, e.StartMs, e.EndMs, e.Count)
}

// FetchAllOrders 获取交易员从 startMs 到 endMs 的所有历史委托（自适应7层细分）
// 返回 *FetchAbortErr 表示获取中断，调用方应保存已获取数据并跳过该交易员
func FetchAllOrders(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	orders, err := client.FetchHistoricalOrders(address, startMs, endMs)
	if err != nil {
		if errors.Is(err, hyperliquid.ErrRateLimited) {
			zap.S().Warnf("[orders] %s: 429 rate limited after retries, skipping trader", address[:10])
			return nil, &FetchAbortErr{Reason: "rate_limited", StartMs: startMs, EndMs: endMs}
		}
		zap.S().Warnf("[orders] probe error for %s: %v", address[:10], err)
		return nil, nil
	}
	if len(orders) == 0 {
		return nil, nil
	}
	if !hyperliquid.IsOrdersAtLimit(orders) {
		return orders, nil
	}

	zap.S().Infof("[orders] %s: hit 2000 limit, splitting by month", address[:10])
	return fetchByMonth(client, address, startMs, endMs, delay)
}

// --- Level 1: 按月 ---
func fetchByMonth(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 1, 0)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] month error %s [%s]: %v", address[:10], cur.Format("2006-01"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s month %s hit limit, split by week", address[:10], cur.Format("2006-01"))
			sub, abortErr := fetchByWeek(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 2: 按周 ---
func fetchByWeek(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 7)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] week error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s week %s hit limit, split by day", address[:10], cur.Format("01-02"))
			sub, abortErr := fetchByDay(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 3: 按天 ---
func fetchByDay(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 1)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] day error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s day %s hit limit, split by hour", address[:10], cur.Format("01-02"))
			sub, abortErr := fetchByHour(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 4: 按小时 ---
func fetchByHour(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(time.Hour)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] hour error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s hour %s hit limit, split by 10min", address[:10], cur.Format("15:04"))
			sub, abortErr := fetchBy10Min(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 5: 按10分钟 ---
func fetchBy10Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(10 * time.Minute)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] 10min error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s 10min %s hit limit, split by 2min", address[:10], cur.Format("15:04"))
			sub, abortErr := fetchBy2Min(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 6: 按2分钟 ---
func fetchBy2Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(2 * time.Minute)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] 2min error %s [%s]: %v", address[:10], cur.Format("15:04:05"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Infof("[orders] %s 2min %s hit limit, split by 30s", address[:10], cur.Format("15:04:05"))
			sub, abortErr := fetchBy30Sec(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, orders...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 7: 按30秒（最细粒度） ---
func fetchBy30Sec(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.OrderEntry, *FetchAbortErr) {
	var all []model.OrderEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(30 * time.Second)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		orders, err := client.FetchHistoricalOrders(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[orders] 30s error %s [%s]: %v", address[:10], cur.Format("15:04:05"), err)
			cur = next
			sleep(delay)
			continue
		}

		all = append(all, orders...)

		if hyperliquid.IsOrdersAtLimit(orders) {
			zap.S().Warnf("[orders] %s 30s %s still at limit (%d), cannot split further — skipping trader",
				address[:10], cur.Format("15:04:05"), len(orders))
			return all, &FetchAbortErr{
				Reason:  "exceeds_limit",
				StartMs: cMs,
				EndMs:   nMs,
				Count:   len(orders),
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
