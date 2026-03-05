package funding

import (
	"errors"
	"fmt"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"go.uber.org/zap"
)

// 5层自适应细分策略：月 → 周 → 天 → 小时 → 10分钟
// 当单次请求返回 >=500 条时，自动向下一层细分
// 429限频时重试3次，仍失败则保存已获取数据并跳过当前交易员

// FetchAbortErr 获取中断错误（429限频重试耗尽）
type FetchAbortErr struct {
	Reason  string // "rate_limited"
	StartMs int64
	EndMs   int64
	Count   int
}

func (e *FetchAbortErr) Error() string {
	return fmt.Sprintf("funding fetch aborted (%s) in window [%d, %d], count %d", e.Reason, e.StartMs, e.EndMs, e.Count)
}

// FetchAllFunding 获取交易员从 startMs 到 endMs 的所有资金费记录（自适应5层细分）
// 返回 *FetchAbortErr 表示获取中断（429限频），调用方应保存已获取数据并跳过该交易员
func FetchAllFunding(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	return fetchByMonth(client, address, startMs, endMs, delay)
}

// --- Level 1: 按月 ---
func fetchByMonth(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	var all []model.FundingEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 1, 0)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		entries, err := client.FetchUserFundingHistory(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[funding] month error %s [%s]: %v", address[:10], cur.Format("2006-01"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsFundingAtLimit(entries) {
			zap.S().Infof("[funding] %s month %s hit limit, split by week", address[:10], cur.Format("2006-01"))
			sub, abortErr := fetchByWeek(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, entries...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 2: 按周 ---
func fetchByWeek(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	var all []model.FundingEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 7)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		entries, err := client.FetchUserFundingHistory(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[funding] week error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsFundingAtLimit(entries) {
			zap.S().Infof("[funding] %s week %s hit limit, split by day", address[:10], cur.Format("01-02"))
			sub, abortErr := fetchByDay(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, entries...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 3: 按天 ---
func fetchByDay(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	var all []model.FundingEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.AddDate(0, 0, 1)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		entries, err := client.FetchUserFundingHistory(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[funding] day error %s [%s]: %v", address[:10], cur.Format("01-02"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsFundingAtLimit(entries) {
			zap.S().Infof("[funding] %s day %s hit limit, split by hour", address[:10], cur.Format("01-02"))
			sub, abortErr := fetchByHour(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, entries...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 4: 按小时 ---
func fetchByHour(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	var all []model.FundingEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(time.Hour)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		entries, err := client.FetchUserFundingHistory(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[funding] hour error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsFundingAtLimit(entries) {
			zap.S().Infof("[funding] %s hour %s hit limit, split by 10min", address[:10], cur.Format("15:04"))
			sub, abortErr := fetchBy10Min(client, address, cMs, nMs, delay)
			all = append(all, sub...)
			if abortErr != nil {
				return all, abortErr
			}
		} else {
			all = append(all, entries...)
		}

		cur = next
		sleep(delay)
	}
	return all, nil
}

// --- Level 5: 按10分钟 ---
func fetchBy10Min(client *hyperliquid.Client, address string, startMs, endMs int64, delay time.Duration) ([]model.FundingEntry, *FetchAbortErr) {
	var all []model.FundingEntry
	cur := time.UnixMilli(startMs).UTC()
	end := time.UnixMilli(endMs).UTC()

	for cur.Before(end) {
		next := cur.Add(10 * time.Minute)
		if next.After(end) {
			next = end
		}
		cMs := cur.UnixMilli()
		nMs := next.UnixMilli()

		entries, err := client.FetchUserFundingHistory(address, cMs, nMs)
		if err != nil {
			if errors.Is(err, hyperliquid.ErrRateLimited) {
				return all, &FetchAbortErr{Reason: "rate_limited", StartMs: cMs, EndMs: nMs}
			}
			zap.S().Warnf("[funding] 10min error %s [%s]: %v", address[:10], cur.Format("15:04"), err)
			cur = next
			sleep(delay)
			continue
		}

		if hyperliquid.IsFundingAtLimit(entries) {
			zap.S().Warnf("[funding] WARNING %s 10min %s still at limit (%d), cannot split further",
				address[:10], cur.Format("15:04"), len(entries))
		}
		all = append(all, entries...)

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
