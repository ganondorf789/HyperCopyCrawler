package follower

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.com/hypercopy/crawler/internal/consts"
	hlclient "github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"github.com/hypercopy/crawler/internal/utility"
	"github.com/redis/go-redis/v9"
	hyperliquid "github.com/sonirico/go-hyperliquid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	wsURL              = "wss://api.hyperliquid.xyz/ws"
	redisKeyAssignment = "addr_dispatch:assignment"
	trackNotifyChannel = "track_wallet_notify"

	pingInterval       = 30 * time.Second
	reconnectBaseDelay = 3 * time.Second
	reconnectMaxDelay  = 60 * time.Second
	subscribeThrottle  = 10 * time.Millisecond

	defaultSlippage = 0.005 // 0.5% market order slippage
)

type wsMsg struct {
	Method       string        `json:"method"`
	Subscription *subscription `json:"subscription,omitempty"`
}

type subscription struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type wsResponse struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type userFillsData struct {
	User       string       `json:"user"`
	Fills      []model.Fill `json:"fills"`
	IsSnapshot bool         `json:"isSnapshot"`
}

type dispatchNotification struct {
	Subscribe   []string `json:"subscribe"`
	Unsubscribe []string `json:"unsubscribe"`
}

type TrackWalletNotifyUser struct {
	UserID int64  `json:"user_id"`
	Remark string `json:"remark"`
	Lang   string `json:"lang"`
}

type TrackWalletNotification struct {
	Users     []TrackWalletNotifyUser `json:"users"`
	Wallet    string                  `json:"wallet"`
	Coin      string                  `json:"coin"`
	Action    string                  `json:"action"` // consts.NotifyAction*: 开仓/平仓/加仓/减仓
	Side      string                  `json:"side"`
	Dir       string                  `json:"dir"`
	Px        string                  `json:"px"`
	Sz        string                  `json:"sz"`
	ClosedPnl string                  `json:"closed_pnl"`
	Time      int64                   `json:"time"`
}

type Follower struct {
	db       *gorm.DB
	rdb      *redis.Client
	hl       *hlclient.Client
	serverIP string

	mu    sync.RWMutex
	addrs map[string]bool

	conn   *websocket.Conn
	connMu sync.Mutex
}

func New(db *gorm.DB, rdb *redis.Client, serverIP string) *Follower {
	return &Follower{
		db:       db,
		rdb:      rdb,
		hl:       hlclient.NewClient(),
		serverIP: serverIP,
		addrs:    make(map[string]bool),
	}
}

func (f *Follower) Run() {
	ctx := context.Background()
	f.loadAddresses(ctx)
	go f.listenDispatch(ctx)
	f.wsLoop(ctx)
}

// ---------- address management ----------

func (f *Follower) loadAddresses(ctx context.Context) {
	all, err := f.rdb.HGetAll(ctx, redisKeyAssignment).Result()
	if err != nil {
		zap.S().Errorf("[follower] redis HGetAll: %v", err)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for addr, ip := range all {
		if ip == f.serverIP {
			f.addrs[addr] = true
		}
	}
	zap.S().Infof("[follower] loaded %d addresses for %s", len(f.addrs), f.serverIP)
}

func (f *Follower) listenDispatch(ctx context.Context) {
	ch := "server:" + f.serverIP + ":dispatch"
	sub := f.rdb.Subscribe(ctx, ch)
	defer sub.Close()

	zap.S().Infof("[follower] listening on %s", ch)

	for msg := range sub.Channel() {
		var n dispatchNotification
		if err := json.Unmarshal([]byte(msg.Payload), &n); err != nil {
			zap.S().Errorf("[follower] decode dispatch: %v", err)
			continue
		}

		f.mu.Lock()
		for _, a := range n.Subscribe {
			f.addrs[a] = true
		}
		for _, a := range n.Unsubscribe {
			delete(f.addrs, a)
		}
		total := len(f.addrs)
		f.mu.Unlock()

		for _, a := range n.Subscribe {
			f.wsSend(wsMsg{
				Method:       "subscribe",
				Subscription: &subscription{Type: "userFills", User: a},
			})
		}
		for _, a := range n.Unsubscribe {
			f.wsSend(wsMsg{
				Method:       "unsubscribe",
				Subscription: &subscription{Type: "userFills", User: a},
			})
		}

		zap.S().Infof("[follower] dispatch: +%d -%d total=%d",
			len(n.Subscribe), len(n.Unsubscribe), total)
	}
}

// ---------- WebSocket ----------

func (f *Follower) wsLoop(ctx context.Context) {
	delay := reconnectBaseDelay
	for {
		if err := f.connectAndServe(ctx); err != nil {
			zap.S().Errorf("[follower] ws error: %v, reconnect in %v", err, delay)
		}
		time.Sleep(delay)
		delay = min(delay*2, reconnectMaxDelay)
	}
}

func (f *Follower) connectAndServe(ctx context.Context) error {
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	f.connMu.Lock()
	f.conn = c
	f.connMu.Unlock()

	defer func() {
		c.Close()
		f.connMu.Lock()
		f.conn = nil
		f.connMu.Unlock()
	}()

	// subscribe all current addresses
	f.mu.RLock()
	addrs := make([]string, 0, len(f.addrs))
	for a := range f.addrs {
		addrs = append(addrs, a)
	}
	f.mu.RUnlock()

	for _, a := range addrs {
		f.wsSend(wsMsg{
			Method:       "subscribe",
			Subscription: &subscription{Type: "userFills", User: a},
		})
		time.Sleep(subscribeThrottle)
	}
	zap.S().Infof("[follower] ws connected, subscribed %d addresses", len(addrs))

	// keep-alive ping
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(pingInterval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				f.wsSend(wsMsg{Method: "ping"})
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	// read loop
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var resp wsResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue
		}

		if resp.Channel == "userFills" {
			f.handleFills(ctx, resp.Data)
		}
	}
}

func (f *Follower) wsSend(msg wsMsg) {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	if f.conn == nil {
		return
	}
	if err := f.conn.WriteJSON(msg); err != nil {
		zap.S().Warnf("[follower] ws write: %v", err)
	}
}

// ---------- fill handling ----------

func (f *Follower) handleFills(ctx context.Context, data json.RawMessage) {
	var d userFillsData
	if err := json.Unmarshal(data, &d); err != nil {
		zap.S().Errorf("[follower] decode fills: %v", err)
		return
	}

	if d.IsSnapshot {
		return
	}

	for i := range d.Fills {
		fill := &d.Fills[i]
		action := classifyAction(fill)
		f.notifyTrackWallets(ctx, d.User, fill, action)
		f.processCopyTrading(ctx, d.User, fill, action)
	}
}

// classifyAction returns consts.NotifyAction* (开仓/平仓/加仓/减仓)
func classifyAction(fill *model.Fill) string {
	isOpen := strings.HasPrefix(fill.Dir, "Open")
	startAbs := new(big.Float).Abs(utility.ParseBigFloatOr0(fill.StartPosition))

	if isOpen {
		if startAbs.Sign() == 0 {
			return consts.NotifyActionOpen
		}
		return consts.NotifyActionIncrease
	}

	if utility.ParseBigFloatOr0(fill.Sz).Cmp(startAbs) >= 0 {
		return consts.NotifyActionClose
	}
	return consts.NotifyActionDecrease
}

// ---------- track wallet notifications ----------

func (f *Follower) notifyTrackWallets(ctx context.Context, addr string, fill *model.Fill, action string) {
	var wallets []model.MyTrackWallet
	if err := f.db.Where("wallet = ? AND status = 1 AND enable_notify = 1", addr).
		Find(&wallets).Error; err != nil {
		zap.S().Errorf("[follower] query track wallets: %v", err)
		return
	}

	var users []TrackWalletNotifyUser
	for _, w := range wallets {
		if w.NotifyAction != "" && !strings.Contains(w.NotifyAction, action) {
			continue
		}
		users = append(users, TrackWalletNotifyUser{
			UserID: w.UserID,
			Remark: w.Remark,
			Lang:   w.Lang,
		})
	}

	if len(users) == 0 {
		return
	}

	n := TrackWalletNotification{
		Users:     users,
		Wallet:    addr,
		Coin:      fill.Coin,
		Action:    action,
		Side:      fill.Side,
		Dir:       fill.Dir,
		Px:        fill.Px,
		Sz:        fill.Sz,
		ClosedPnl: fill.ClosedPnl,
		Time:      fill.Time,
	}
	payload, _ := json.Marshal(n)
	if err := f.rdb.Publish(ctx, trackNotifyChannel, string(payload)).Err(); err != nil {
		zap.S().Errorf("[follower] publish notify: %v", err)
	} else {
		zap.S().Infof("[follower] notified %d users %s %s %s %s@%s",
			len(users), utility.AbbrWithEllipsis(addr), fill.Dir, fill.Sz, fill.Coin, fill.Px)
	}
}

// ---------- copy trading ----------

func (f *Follower) processCopyTrading(ctx context.Context, addr string, fill *model.Fill, action string) {
	var configs []model.CopyTradingConfig
	if err := f.db.Where("target_wallet = ? AND follow_type = ? AND status = 1", addr, consts.FollowTypeAuto).
		Find(&configs).Error; err != nil {
		zap.S().Errorf("[follower] query copy configs: %v", err)
		return
	}

	for _, cfg := range configs {
		if !utility.SymbolAllowed(cfg.SymbolList, cfg.SymbolListType, fill.Coin) {
			continue
		}
		if cfg.FollowSymbol != "" && cfg.FollowSymbol != fill.Coin {
			continue
		}

		if action == consts.NotifyActionIncrease && cfg.OptFollowupIncrease == 0 && cfg.OptPositionIncreaseOpening == 0 {
			continue
		}
		if action == consts.NotifyActionDecrease && cfg.OptFollowupDecrease == 0 {
			continue
		}

		// follow-once: skip if active position already exists
		if cfg.FollowOnce == 1 {
			var cnt int64
			f.db.Model(&model.CopyTrading{}).
				Where("copy_trading_id = ? AND status IN ?", cfg.ID,
					[]string{model.CopyTradingStatusNotStarted, model.CopyTradingStatusFollowing}).
				Count(&cnt)
			if cnt > 0 {
				continue
			}
		}

		f.executeCopyOrder(ctx, cfg, addr, fill, action)
	}
}

func (f *Follower) executeCopyOrder(ctx context.Context, cfg model.CopyTradingConfig, addr string, fill *model.Fill, action string) {
	var wallet model.Wallet
	if err := f.db.Where("user_id = ? AND address = ? AND status = 1", cfg.UserID, cfg.MainWallet).
		First(&wallet).Error; err != nil {
		zap.S().Warnf("[follower] wallet not found: user=%d addr=%s: %v", cfg.UserID, utility.AbbrWithEllipsis(cfg.MainWallet), err)
		f.saveRecord(cfg, fill, consts.ExecStatusSkipped, "wallet not found")
		return
	}

	isBuy, orderSize, orderPrice, reduceOnly, err := f.calcOrderParams(ctx, cfg, &wallet, addr, fill, action)
	if err != nil {
		zap.S().Warnf("[follower] calc order: user=%d cfg=%d: %v", cfg.UserID, cfg.ID, err)
		f.saveRecord(cfg, fill, consts.ExecStatusSkipped, err.Error())
		return
	}

	exchange, err := f.newExchange(ctx, &wallet)
	if err != nil {
		zap.S().Errorf("[follower] exchange init: user=%d: %v", cfg.UserID, err)
		f.saveRecord(cfg, fill, consts.ExecStatusFailed, err.Error())
		return
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       fill.Coin,
		IsBuy:      isBuy,
		Price:      orderPrice,
		Size:       orderSize,
		ReduceOnly: reduceOnly,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifIoc},
		},
	}

	status, err := exchange.Order(ctx, req, nil)
	if err != nil {
		zap.S().Errorf("[follower] order failed: user=%d cfg=%d %s %.6f %s@%.2f: %v",
			cfg.UserID, cfg.ID, fill.Coin, orderSize, boolToSide(isBuy), orderPrice, err)
		f.saveCopyTradingAndRecord(cfg, addr, fill, action, model.CopyTradingStatusFailed, consts.ExecStatusFailed, err.Error())
		return
	}

	if status.Error != nil {
		zap.S().Errorf("[follower] order rejected: user=%d cfg=%d: %s", cfg.UserID, cfg.ID, *status.Error)
		f.saveCopyTradingAndRecord(cfg, addr, fill, action, model.CopyTradingStatusFailed, consts.ExecStatusFailed, *status.Error)
		return
	}

	zap.S().Infof("[follower] order placed: user=%d cfg=%d %s %.6f %s@%.2f",
		cfg.UserID, cfg.ID, fill.Coin, orderSize, boolToSide(isBuy), orderPrice)
	f.saveCopyTradingAndRecord(cfg, addr, fill, action, model.CopyTradingStatusFollowing, consts.ExecStatusSuccess, "")
}

func (f *Follower) newExchange(ctx context.Context, w *model.Wallet) (*hyperliquid.Exchange, error) {
	privateKey, err := crypto.HexToECDSA(w.APISecretKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return hyperliquid.NewExchange(
		ctx,
		privateKey,
		hyperliquid.MainnetAPIURL,
		nil,
		"",
		w.Address,
		nil,
		nil,
	), nil
}

// calcOrderParams returns (isBuy, size, price, reduceOnly, error).
func (f *Follower) calcOrderParams(ctx context.Context, cfg model.CopyTradingConfig, wallet *model.Wallet, traderAddr string, fill *model.Fill, action string) (bool, float64, float64, bool, error) {
	px, err := strconv.ParseFloat(fill.Px, 64)
	if err != nil || px <= 0 {
		return false, 0, 0, false, fmt.Errorf("invalid fill price: %s", fill.Px)
	}
	fillSz, _ := strconv.ParseFloat(fill.Sz, 64)
	startPos, _ := strconv.ParseFloat(fill.StartPosition, 64)

	isClose := action == consts.NotifyActionClose || action == consts.NotifyActionDecrease
	isFollowUp := action == consts.NotifyActionIncrease || action == consts.NotifyActionDecrease

	isBuy := strings.Contains(fill.Dir, "Long")
	if isClose {
		isBuy = !isBuy
	}

	modelValue, _ := strconv.ParseFloat(cfg.FollowModelValue, 64)

	var orderSize float64

	switch cfg.FollowModel {
	case consts.FollowModelAssetProportional:
		ratio := modelValue / 100
		orderSize, err = f.calcAssetProportional(ctx, traderAddr, wallet.Address, px, fillSz, ratio)
		if err != nil {
			return false, 0, 0, false, err
		}

	case consts.FollowModelPositionProportional:
		ratio := modelValue / 100
		orderSize = fillSz * ratio

	case consts.FollowModelFixedValue:
		if isFollowUp && math.Abs(startPos) > 0 {
			posRatio := fillSz / math.Abs(startPos)
			followerPos := f.getFollowerPositionSize(wallet.Address, fill.Coin)
			orderSize = math.Abs(followerPos) * posRatio
		} else {
			orderSize = modelValue / px
		}

	default:
		return false, 0, 0, false, fmt.Errorf("unsupported follow_model: %d", cfg.FollowModel)
	}

	orderValueUSD := orderSize * px
	minVal, _ := strconv.ParseFloat(cfg.MinValue, 64)
	maxVal, _ := strconv.ParseFloat(cfg.MaxValue, 64)

	if minVal > 0 && orderValueUSD < minVal {
		return false, 0, 0, false, fmt.Errorf("order value %.2f below min %.2f", orderValueUSD, minVal)
	}
	if maxVal > 0 && orderValueUSD > maxVal {
		orderSize = maxVal / px
	}

	slippage := defaultSlippage
	orderPrice := px
	if isBuy {
		orderPrice = px * (1 + slippage)
	} else {
		orderPrice = px * (1 - slippage)
	}

	orderPrice = math.Round(orderPrice*1e6) / 1e6
	orderSize = math.Round(orderSize*1e6) / 1e6

	if orderSize <= 0 {
		return false, 0, 0, false, fmt.Errorf("calculated size is zero")
	}

	return isBuy, orderSize, orderPrice, isClose, nil
}

// calcAssetProportional: 资产等比 — 目标用了 X% 本金，跟单也用 X% 本金。
func (f *Follower) calcAssetProportional(ctx context.Context, traderAddr, followerAddr string, px, fillSz, multiplier float64) (float64, error) {
	traderState, err := f.hl.FetchClearinghouseState(traderAddr)
	if err != nil {
		return 0, fmt.Errorf("fetch trader state: %w", err)
	}
	traderAV, _ := strconv.ParseFloat(traderState.MarginSummary.AccountValue, 64)
	if traderAV <= 0 {
		return 0, fmt.Errorf("trader account value is zero")
	}

	followerState, err := f.hl.FetchClearinghouseState(followerAddr)
	if err != nil {
		return 0, fmt.Errorf("fetch follower state: %w", err)
	}
	followerAV, _ := strconv.ParseFloat(followerState.MarginSummary.AccountValue, 64)
	if followerAV <= 0 {
		return 0, fmt.Errorf("follower account value is zero")
	}

	traderOrderValue := px * fillSz
	capitalRatio := traderOrderValue / traderAV
	followerOrderValue := followerAV * capitalRatio * multiplier

	return followerOrderValue / px, nil
}

// getFollowerPositionSize returns the follower's current position size for a coin (signed).
func (f *Follower) getFollowerPositionSize(followerAddr, coin string) float64 {
	state, err := f.hl.FetchClearinghouseState(followerAddr)
	if err != nil {
		zap.S().Warnf("[follower] fetch position for %s: %v", utility.AbbrWithEllipsis(followerAddr), err)
		return 0
	}
	for _, ap := range state.AssetPositions {
		if ap.Position.Coin == coin {
			sz, _ := strconv.ParseFloat(ap.Position.Szi, 64)
			return sz
		}
	}
	return 0
}

func (f *Follower) saveCopyTradingAndRecord(cfg model.CopyTradingConfig, addr string, fill *model.Fill, action string, ctStatus string, execStatus int, errMsg string) {
	traderSzi := utility.CalcResultSzi(fill.StartPosition, fill.Sz, fill.Side)
	posValue := utility.MulStr(fill.Px, fill.Sz)

	cp := model.CopyTrading{
		CopyTradingID:                  cfg.ID,
		UserID:                         cfg.UserID,
		TargetWallet:                   cfg.TargetWallet,
		TargetWalletPlatform:           cfg.TargetWalletPlatform,
		Remark:                         cfg.Remark,
		FollowType:                     cfg.FollowType,
		FollowOnce:                     cfg.FollowOnce,
		PositionConditions:             cfg.PositionConditions,
		TraderConditions:               cfg.TraderConditions,
		TagAccountValue:                cfg.TagAccountValue,
		TagProfitScale:                 cfg.TagProfitScale,
		TagDirection:                   cfg.TagDirection,
		TagTradingRhythm:               cfg.TagTradingRhythm,
		TagProfitStatus:                cfg.TagProfitStatus,
		TagTradingStyles:               cfg.TagTradingStyles,
		TraderMetricPeriod:             cfg.TraderMetricPeriod,
		FollowMarginMode:               cfg.FollowMarginMode,
		FollowSymbol:                   cfg.FollowSymbol,
		Leverage:                       cfg.Leverage,
		MarginMode:                     cfg.MarginMode,
		FollowModel:                    cfg.FollowModel,
		FollowModelValue:               cfg.FollowModelValue,
		MinValue:                       cfg.MinValue,
		MaxValue:                       cfg.MaxValue,
		MaxMarginUsage:                 cfg.MaxMarginUsage,
		TpValue:                        cfg.TpValue,
		SlValue:                        cfg.SlValue,
		OptFollowupDecrease:            cfg.OptFollowupDecrease,
		OptFollowupIncrease:            cfg.OptFollowupIncrease,
		OptForcedLiquidationProtection: cfg.OptForcedLiquidationProtection,
		OptPositionIncreaseOpening:     cfg.OptPositionIncreaseOpening,
		OptSlippageProtection:          cfg.OptSlippageProtection,
		SymbolListType:                 cfg.SymbolListType,
		SymbolList:                     cfg.SymbolList,
		MainWallet:                     cfg.MainWallet,
		MainWalletPlatform:             cfg.MainWalletPlatform,
		CopyTradingStatus:              cfg.Status,
		CopyTradingCreatedAt:           cfg.CreatedAt,
		CopyTradingUpdatedAt:           cfg.UpdatedAt,
		TraderAddress:                  addr,
		TraderCoin:                     fill.Coin,
		TraderSzi:                      traderSzi,
		TraderEntryPx:                  fill.Px,
		TraderPositionValue:            posValue,
		Status:                         ctStatus,
		ErrorMsg:                       errMsg,
	}

	if err := f.db.Create(&cp).Error; err != nil {
		zap.S().Errorf("[follower] create copy_trading: %v", err)
	}

	f.saveRecord(cfg, fill, execStatus, errMsg)
}

func (f *Follower) saveRecord(cfg model.CopyTradingConfig, fill *model.Fill, execStatus int, errMsg string) {
	r := model.CopyTradeRecord{
		UserID:        cfg.UserID,
		Address:       cfg.TargetWallet,
		Coin:          fill.Coin,
		Direction:     fill.Dir,
		Size:          fill.Sz,
		Price:         fill.Px,
		ClosedPnl:     fill.ClosedPnl,
		ExecuteStatus: execStatus,
		ErrorMsg:      errMsg,
		TradeTime:     time.UnixMilli(fill.Time),
	}
	if err := f.db.Create(&r).Error; err != nil {
		zap.S().Errorf("[follower] create record: %v", err)
	}
}

func boolToSide(isBuy bool) string {
	if isBuy {
		return "BUY"
	}
	return "SELL"
}
