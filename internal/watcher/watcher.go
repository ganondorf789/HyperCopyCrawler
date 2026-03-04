package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/proxy"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	redisChannel        = "new_positions"
	marketAlertChannel  = "market_alert"
	redisTimelineKey    = "watcher:new_position_timeline"
)

type NewPositionEvent struct {
	Address               string `json:"address"`
	Coin                  string `json:"coin"`
	Szi                   string `json:"szi"`
	LeverageType          string `json:"leverageType"`
	Leverage              int    `json:"leverage"`
	EntryPx               string `json:"entryPx"`
	PositionValue         string `json:"positionValue"`
	UnrealizedPnl         string `json:"unrealizedPnl"`
	ReturnOnEquity        string `json:"returnOnEquity"`
	LiquidationPx         string `json:"liquidationPx"`
	MarginUsed            string `json:"marginUsed"`
	MaxLeverage           int    `json:"maxLeverage"`
	CumFundingAllTime     string `json:"cumFundingAllTime"`
	CumFundingSinceOpen   string `json:"cumFundingSinceOpen"`
	CumFundingSinceChange string `json:"cumFundingSinceChange"`
}

type Watcher struct {
	db       *gorm.DB
	rdb      *redis.Client
	proxyMgr *proxy.Manager
	rate     int
	offset   int
	limit    int
}

func New(db *gorm.DB, rdb *redis.Client, proxyMgr *proxy.Manager, rate, offset, limit int) *Watcher {
	return &Watcher{
		db:       db,
		rdb:      rdb,
		proxyMgr: proxyMgr,
		rate:     rate,
		offset:   offset,
		limit:    limit,
	}
}

func (w *Watcher) Run() {
	interval := time.Second / time.Duration(w.rate)
	client := w.newClient()

	for round := 1; ; round++ {
		var setting model.SystemSetting
		if err := w.db.First(&setting).Error; err != nil {
			zap.S().Warnf("[watcher] load system setting error: %v, using defaults (5min / 3 positions)", err)
			setting = model.SystemSetting{MarketMinutes: 5, MarketNewPositionCount: 3}
		}

		var leaders []model.Leaderboard
		query := w.db.Order("vlm DESC")
		if w.offset > 0 {
			query = query.Offset(w.offset)
		}
		if w.limit > 0 {
			query = query.Limit(w.limit)
		}
		if err := query.Find(&leaders).Error; err != nil {
			zap.S().Errorf("[watcher] load leaderboard error: %v, retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if len(leaders) == 0 {
			zap.S().Warn("[watcher] no traders in leaderboard, retrying in 30s")
			time.Sleep(30 * time.Second)
			continue
		}

		addresses := make([]string, len(leaders))
		for i, l := range leaders {
			addresses[i] = l.EthAddress
		}

		holdingMap, err := w.loadHoldings(addresses)
		if err != nil {
			zap.S().Errorf("[watcher] load holdings error: %v, retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		total := len(addresses)
		zap.S().Infof("[watcher] round %d: %d traders to watch (offset=%d, limit=%d, rate=%d/s, market=%dmin/%d)",
			round, total, w.offset, w.limit, w.rate, setting.MarketMinutes, setting.MarketNewPositionCount)

		succeeded, failed := 0, 0
		for i, address := range addresses {
			start := time.Now()

			oldCoins := holdingMap[address]
			if err := w.processOne(client, address, oldCoins, &setting); err != nil {
				zap.S().Warnf("[watcher] %s error: %v", address[:10], err)
				failed++
			} else {
				succeeded++
			}

			if (i+1)%100 == 0 || i+1 == total {
				zap.S().Infof("[watcher] progress: %d/%d", i+1, total)
			}

			elapsed := time.Since(start)
			if elapsed < interval {
				time.Sleep(interval - elapsed)
			}
		}

		zap.S().Infof("[watcher] round %d done: %d/%d succeeded, %d errors",
			round, succeeded, total, failed)
	}
}

func (w *Watcher) newClient() *hyperliquid.Client {
	p := w.proxyMgr.Next()
	if p == nil {
		return hyperliquid.NewClient()
	}
	c, err := hyperliquid.NewClientWithProxy(proxy.ProxyURL(p))
	if err != nil {
		zap.S().Warnf("[watcher] create proxy client error: %v, using direct", err)
		return hyperliquid.NewClient()
	}
	return c
}

func (w *Watcher) loadHoldings(addresses []string) (map[string]map[string]string, error) {
	var holdings []model.TraderCoinHolding
	if err := w.db.Where("address IN ?", addresses).Find(&holdings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string, len(holdings))
	for _, h := range holdings {
		coinMap := make(map[string]string, len(h.Positions))
		for _, p := range h.Positions {
			coinMap[p.Coin] = p.Szi
		}
		result[h.Address] = coinMap
	}
	return result, nil
}

func (w *Watcher) processOne(client *hyperliquid.Client, address string, oldCoins map[string]string, setting *model.SystemSetting) error {
	chState, err := client.FetchClearinghouseState(address)
	if err != nil {
		return err
	}

	var newEvents []NewPositionEvent
	currentPositions := make(model.CoinPositions, 0, len(chState.AssetPositions))

	for _, ap := range chState.AssetPositions {
		p := ap.Position
		currentPositions = append(currentPositions, model.CoinPosition{
			Coin: p.Coin,
			Szi:  p.Szi,
		})

		if _, exists := oldCoins[p.Coin]; !exists {
			liqPx := ""
			if p.LiquidationPx != nil {
				liqPx = *p.LiquidationPx
			}
			newEvents = append(newEvents, NewPositionEvent{
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
	}

	sort.Slice(currentPositions, func(i, j int) bool {
		return currentPositions[i].Coin < currentPositions[j].Coin
	})

	if err := w.upsertHolding(address, currentPositions); err != nil {
		return fmt.Errorf("upsert holding: %w", err)
	}

	for _, evt := range newEvents {
		w.trackAndPublish(evt, setting)
	}

	return nil
}

func (w *Watcher) upsertHolding(address string, positions model.CoinPositions) error {
	holding := model.TraderCoinHolding{
		Address:   address,
		Positions: positions,
	}
	return w.db.Where("address = ?", address).
		Assign(model.TraderCoinHolding{Positions: positions}).
		FirstOrCreate(&holding).Error
}

func (w *Watcher) trackAndPublish(evt NewPositionEvent, setting *model.SystemSetting) {
	ctx := context.Background()

	data, err := json.Marshal(evt)
	if err != nil {
		zap.S().Errorf("[watcher] marshal event error: %v", err)
		return
	}
	if err := w.rdb.Publish(ctx, redisChannel, string(data)).Err(); err != nil {
		zap.S().Errorf("[watcher] redis publish new position error: %v", err)
	}

	now := float64(time.Now().UnixMilli())
	member := fmt.Sprintf("%s:%s:%d", evt.Address, evt.Coin, int64(now))

	w.rdb.ZAdd(ctx, redisTimelineKey, redis.Z{Score: now, Member: member})

	windowStart := now - float64(setting.MarketMinutes)*60*1000
	w.rdb.ZRemRangeByScore(ctx, redisTimelineKey, "-inf", fmt.Sprintf("%f", windowStart))

	count, err := w.rdb.ZCard(ctx, redisTimelineKey).Result()
	if err != nil {
		zap.S().Errorf("[watcher] redis zcard error: %v", err)
	}

	zap.S().Infof("[watcher] new position: %s %s szi=%s entry=%s (count=%d/%d in %dmin)",
		evt.Address[:10], evt.Coin, evt.Szi, evt.EntryPx,
		count, setting.MarketNewPositionCount, setting.MarketMinutes)

	if count >= int64(setting.MarketNewPositionCount) {
		w.publishMarketAlert(count, setting)
	}
}

type MarketAlert struct {
	Count    int64 `json:"count"`
	Minutes  int   `json:"minutes"`
	Threshold int  `json:"threshold"`
}

func (w *Watcher) publishMarketAlert(count int64, setting *model.SystemSetting) {
	alert := MarketAlert{
		Count:     count,
		Minutes:   setting.MarketMinutes,
		Threshold: setting.MarketNewPositionCount,
	}
	data, err := json.Marshal(alert)
	if err != nil {
		zap.S().Errorf("[watcher] marshal market alert error: %v", err)
		return
	}

	if err := w.rdb.Publish(context.Background(), marketAlertChannel, string(data)).Err(); err != nil {
		zap.S().Errorf("[watcher] redis publish market alert error: %v", err)
		return
	}

	zap.S().Warnf("[watcher] MARKET ALERT: %d new positions in %d minutes (threshold=%d)",
		count, setting.MarketMinutes, setting.MarketNewPositionCount)
}
