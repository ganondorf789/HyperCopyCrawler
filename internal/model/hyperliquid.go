package model

import "encoding/json"

// --- Leaderboard ---

type LeaderboardResponse struct {
	LeaderboardRows []LeaderboardRow `json:"leaderboardRows"`
}

type LeaderboardRow struct {
	EthAddress         string              `json:"ethAddress"`
	AccountValue       string              `json:"accountValue"`
	DisplayName        string              `json:"displayName"`
	Prize              json.RawMessage     `json:"prize"`
	WindowPerformances []WindowPerformance `json:"windowPerformances"`
}

// WindowPerformance 是 [window, {pnl, roi, vlm}] 的二元组
type WindowPerformance [2]json.RawMessage

type PerformanceData struct {
	Pnl string `json:"pnl"`
	Roi string `json:"roi"`
	Vlm string `json:"vlm"`
}

func (wp WindowPerformance) Parse() (window string, data PerformanceData, err error) {
	if err = json.Unmarshal(wp[0], &window); err != nil {
		return
	}
	err = json.Unmarshal(wp[1], &data)
	return
}

// --- Portfolio ---

type PortfolioRequest struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type PortfolioResponse struct {
	AccountValueHistory []TimeSeriesEntry `json:"accountValueHistory"`
	PnlHistory          []TimeSeriesEntry `json:"pnlHistory"`
}

type TimeSeriesEntry = json.RawMessage

// --- UserFillsByTime ---

type FillsByTimeRequest struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// Fill API 返回的成交记录
type Fill struct {
	Coin          string `json:"coin"`
	Px            string `json:"px"`
	Sz            string `json:"sz"`
	Side          string `json:"side"`
	Time          int64  `json:"time"`
	StartPosition string `json:"startPosition"`
	Dir           string `json:"dir"`
	ClosedPnl     string `json:"closedPnl"`
	Hash          string `json:"hash"`
	Oid           int64  `json:"oid"`
	Crossed       bool   `json:"crossed"`
	Fee           string `json:"fee"`
	Tid           int64  `json:"tid"`
	Cloid         string `json:"cloid"`
	FeeToken      string `json:"feeToken"`
}

// --- UserFundingHistory ---

type FundingHistoryRequest struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// FundingEntry API 返回的资金费记录（userFunding 接口）
type FundingEntry struct {
	Time  int64        `json:"time"`
	Hash  string       `json:"hash"`
	Delta FundingDelta `json:"delta"`
}

type FundingDelta struct {
	Type        string `json:"type"`
	Coin        string `json:"coin"`
	Usdc        string `json:"usdc"`
	Szi         string `json:"szi"`
	FundingRate string `json:"fundingRate"`
	NSamples    *int   `json:"nSamples"`
}

// --- HistoricalOrders ---

type HistoricalOrdersRequest struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// OrderEntry API 返回的历史委托记录
type OrderEntry struct {
	Coin             string          `json:"coin"`
	Side             string          `json:"side"`
	LimitPx          string          `json:"limitPx"`
	Sz               string          `json:"sz"`
	Oid              int64           `json:"oid"`
	Timestamp        int64           `json:"timestamp"`
	TriggerCondition string          `json:"triggerCondition"`
	IsTrigger        bool            `json:"isTrigger"`
	TriggerPx        string          `json:"triggerPx"`
	Children         json.RawMessage `json:"children"`
	IsPositionTpsl   bool            `json:"isPositionTpsl"`
	ReduceOnly       bool            `json:"reduceOnly"`
	OrderType        string          `json:"orderType"`
	OrigSz           string          `json:"origSz"`
	Tif              string          `json:"tif"`
	Cloid            *string         `json:"cloid"`
	Status           string          `json:"status"`
}

// --- ClearinghouseState (永续合约持仓 + 保证金) ---

type ClearinghouseState struct {
	AssetPositions             []AssetPosition `json:"assetPositions"`
	CrossMarginSummary         MarginSummary   `json:"crossMarginSummary"`
	MarginSummary              MarginSummary   `json:"marginSummary"`
	Withdrawable               string          `json:"withdrawable"`
	CrossMaintenanceMarginUsed string          `json:"crossMaintenanceMarginUsed"`
	Time                       int64           `json:"time"`
}

type AssetPosition struct {
	Type     string   `json:"type"`
	Position Position `json:"position"`
}

type Position struct {
	Coin           string     `json:"coin"`
	Szi            string     `json:"szi"`
	Leverage       Leverage   `json:"leverage"`
	EntryPx        string     `json:"entryPx"`
	PositionValue  string     `json:"positionValue"`
	UnrealizedPnl  string     `json:"unrealizedPnl"`
	ReturnOnEquity string     `json:"returnOnEquity"`
	LiquidationPx  *string    `json:"liquidationPx"`
	MarginUsed     string     `json:"marginUsed"`
	MaxLeverage    int        `json:"maxLeverage"`
	CumFunding     CumFunding `json:"cumFunding"`
}

type Leverage struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type CumFunding struct {
	AllTime     string `json:"allTime"`
	SinceOpen   string `json:"sinceOpen"`
	SinceChange string `json:"sinceChange"`
}

type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUsd     string `json:"totalRawUsd"`
	TotalMarginUsed string `json:"totalMarginUsed"`
}

// --- SpotClearinghouseState (现货持仓) ---

type SpotClearinghouseState struct {
	Balances []SpotBalance `json:"balances"`
}

type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Total    string `json:"total"`
	Hold     string `json:"hold"`
	EntryNtl string `json:"entryNtl"`
}
