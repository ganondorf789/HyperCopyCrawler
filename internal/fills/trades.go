package fills

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"

	"github.com/hypercopy/crawler/internal/model"
	"gorm.io/gorm"
)

// BuildCompletedTrades 从 fills 重建交易员的 completed trades
// 逻辑：按 (coin) 分组，按时间排序，跟踪仓位变化：
//   - "Open Long" / "Open Short" → 开仓/加仓
//   - "Close Long" / "Close Short" → 减仓/平仓
//   - 当仓位归零时，一笔 completed trade 完成
func BuildCompletedTrades(db *gorm.DB, address string) error {
	// 1. 加载该交易员所有 fills，按时间排序
	var allFills []model.TraderFill
	if err := db.Where("address = ?", address).Order("time ASC").Find(&allFills).Error; err != nil {
		return fmt.Errorf("load fills: %w", err)
	}
	if len(allFills) == 0 {
		return nil
	}

	// 2. 按 coin 分组
	grouped := make(map[string][]model.TraderFill)
	for _, f := range allFills {
		grouped[f.Coin] = append(grouped[f.Coin], f)
	}

	// 3. 对每个 coin 构建 completed trades
	var trades []model.CompletedTrade
	for coin, coinFills := range grouped {
		sort.Slice(coinFills, func(i, j int) bool {
			return coinFills[i].Time < coinFills[j].Time
		})
		ct := buildForCoin(address, coin, coinFills)
		trades = append(trades, ct...)
	}

	// 4. 事务：删除旧记录，写入新记录
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("address = ?", address).Delete(&model.CompletedTrade{}).Error; err != nil {
			return fmt.Errorf("delete old completed trades: %w", err)
		}
		if len(trades) == 0 {
			return nil
		}
		// 分批写入
		batch := 500
		for i := 0; i < len(trades); i += batch {
			end := i + batch
			if end > len(trades) {
				end = len(trades)
			}
			if err := tx.Create(trades[i:end]).Error; err != nil {
				return fmt.Errorf("insert completed trades batch %d: %w", i/batch, err)
			}
		}
		log.Printf("[trades] %s: rebuilt %d completed trades from %d fills", address[:10], len(trades), len(allFills))
		return nil
	})
}

// positionState 跟踪单个 coin 的仓位状态
type positionState struct {
	direction  string  // "long" / "short"
	marginMode string  // "isolated" / "cross" (从 startPosition 推断)
	size       float64 // 当前持仓量（绝对值）
	maxSize    float64 // 最大持仓量
	costBasis  float64 // 累计开仓成本 (price * size)
	openSize   float64 // 累计开仓数量
	closeValue float64 // 累计平仓价值 (price * size)
	closeSize  float64 // 累计平仓数量
	totalFee   float64 // 累计手续费
	pnl        float64 // 累计 closedPnl
	startTime  int64   // 第一笔开仓时间
	endTime    int64   // 最后一笔平仓时间
	fillCount  int     // 成交笔数
}

func buildForCoin(address, coin string, fills []model.TraderFill) []model.CompletedTrade {
	var result []model.CompletedTrade
	var state *positionState

	for _, f := range fills {
		px := parseFloat(f.Px)
		sz := parseFloat(f.Sz)
		fee := parseFloat(f.Fee)
		closedPnl := parseFloat(f.ClosedPnl)

		isOpen := f.Dir == "Open Long" || f.Dir == "Open Short"
		isClose := f.Dir == "Close Long" || f.Dir == "Close Short"

		if !isOpen && !isClose {
			// 未知方向，跳过
			continue
		}

		if isOpen {
			dir := "long"
			if f.Dir == "Open Short" {
				dir = "short"
			}

			if state == nil {
				// 新开仓
				state = &positionState{
					direction: dir,
					startTime: f.Time,
				}
			}

			state.size += sz
			if state.size > state.maxSize {
				state.maxSize = state.size
			}
			state.costBasis += px * sz
			state.openSize += sz
			state.totalFee += fee
			state.pnl += closedPnl
			state.fillCount++
		}

		if isClose {
			if state == nil {
				// 没有对应的开仓记录，跳过
				continue
			}

			state.size -= sz
			state.closeValue += px * sz
			state.closeSize += sz
			state.totalFee += fee
			state.pnl += closedPnl
			state.endTime = f.Time
			state.fillCount++

			// 仓位归零（或接近零），完成一笔交易
			if state.size <= 1e-12 {
				trade := model.CompletedTrade{
					Address:    address,
					Coin:       coin,
					MarginMode: detectMarginMode(fills),
					Direction:  state.direction,
					Size:       roundTo(state.maxSize, 8),
					EntryPrice: roundTo(safeDivide(state.costBasis, state.openSize), 8),
					ClosePrice: roundTo(safeDivide(state.closeValue, state.closeSize), 8),
					StartTime:  state.startTime,
					EndTime:    state.endTime,
					TotalFee:   roundTo(state.totalFee, 6),
					Pnl:        roundTo(state.pnl, 6),
					FillCount:  state.fillCount,
				}
				result = append(result, trade)
				state = nil
			}
		}
	}

	return result
}

// detectMarginMode 从 fills 的 crossed 字段推断保证金模式
func detectMarginMode(fills []model.TraderFill) string {
	for _, f := range fills {
		if f.Crossed {
			return "cross"
		}
	}
	return "isolated"
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func roundTo(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
