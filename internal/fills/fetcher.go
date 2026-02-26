package fills

import (
	"log"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
)

// 5层自适应细分策略（从后往前）：月 → 周 → 天 → 小时 → 10分钟
// 当单次请求返回 >=2000 条时，自动向下一层细分

// FetchAllFills 获取交易员从 startMs 到 endMs 的所有成交记录（自适应5层细分）
func FetchAllFills(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	// 先探测全量
	fills, err := client.FetchUserFillsByTime(address, startMs, endMs)
	if err != nil {
		log.Printf("[fills] probe error for %s: %v", address[:10], err)
		return nil
	}
	if len(fills) == 0 {
		return nil
	}
	// 未达上限，直接返回
	if !hyperliquid.IsAtLimit(fills) {
		return fills
	}

	// 达到上限，按月细分
	log.Printf("[fills] %s: hit 2000 limit, splitting by month", address[:10])
	return fetchByMonth(client, address, startMs, endMs, delay)
}

// --- Level 1: 按月 ---
func fetchByMonth(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	var all []hyperliquid.Fill
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
			log.Printf("[fills] month error %s [%s]: %v", address[:10], cur.Format("2006-01"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			log.Printf("[fills] %s month %s hit limit, split by week", address[:10], cur.Format("2006-01"))
			all = append(all, fetchByWeek(client, address, cMs, nMs, delay)...)
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all
}

// --- Level 2: 按周 ---
func fetchByWeek(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	var all []hyperliquid.Fill
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
			log.Printf("[fills] week error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			log.Printf("[fills] %s week %s hit limit, split by day", address[:10], cur.Format("01-02"))
			all = append(all, fetchByDay(client, address, cMs, nMs, delay)...)
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all
}

// --- Level 3: 按天 ---
func fetchByDay(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	var all []hyperliquid.Fill
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
			log.Printf("[fills] day error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			log.Printf("[fills] %s day %s hit limit, split by hour", address[:10], cur.Format("01-02"))
			all = append(all, fetchByHour(client, address, cMs, nMs, delay)...)
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all
}

// --- Level 4: 按小时 ---
func fetchByHour(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	var all []hyperliquid.Fill
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
			log.Printf("[fills] hour error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			log.Printf("[fills] %s hour %s hit limit, split by 10min", address[:10], cur.Format("15:04"))
			all = append(all, fetchBy10Min(client, address, cMs, nMs, delay)...)
		} else {
			all = append(all, fills...)
		}

		cur = next
		sleep(delay)
	}
	return all
}

// --- Level 5: 按10分钟 ---
func fetchBy10Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) []hyperliquid.Fill {
	var all []hyperliquid.Fill
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
			log.Printf("[fills] 10min error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsAtLimit(fills) {
			log.Printf("[fills] WARNING %s 10min %s still at limit (%d), cannot split further",
				address[:10], cur.Format("15:04"), len(fills))
		}
		all = append(all, fills...)

		cur = next
		sleep(delay)
	}
	return all
}

func sleep(d time.Duration) {
	if d > 0 {
		time.Sleep(d)
	}
}
