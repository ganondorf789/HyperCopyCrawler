package hyperliquid

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	leaderboardURL = "https://stats-data.hyperliquid.xyz/Mainnet/leaderboard"
	infoURL        = "https://api.hyperliquid.xyz/info"

	fillsLimit = 2000 // API 单次返回上限
)

// Client Hyperliquid API 客户端
type Client struct {
	http *http.Client
}

// NewClient 创建无代理客户端
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// NewClientWithProxy 创建带代理的客户端
func NewClientWithProxy(proxyURL string) (*Client, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}
	transport := &http.Transport{
		Proxy:           http.ProxyURL(u),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	return &Client{
		http: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}, nil
}

// --- Leaderboard ---

type LeaderboardResponse struct {
	LeaderboardRows []LeaderboardRow `json:"leaderboardRows"`
}

type LeaderboardRow struct {
	EthAddress         string              `json:"ethAddress"`
	AccountValue       string              `json:"accountValue"`
	DisplayName        string              `json:"displayName"`
	Prize              string              `json:"prize"`
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

func (c *Client) FetchLeaderboard() (*LeaderboardResponse, error) {
	resp, err := c.http.Get(leaderboardURL)
	if err != nil {
		return nil, fmt.Errorf("fetch leaderboard: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read leaderboard body: %w", err)
	}

	var result LeaderboardResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal leaderboard: %w", err)
	}
	return &result, nil
}

// --- Portfolio (accountValueHistory + pnlHistory) ---

type PortfolioRequest struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type PortfolioResponse struct {
	AccountValueHistory []TimeSeriesEntry `json:"accountValueHistory"`
	PnlHistory          []TimeSeriesEntry `json:"pnlHistory"`
}

type TimeSeriesEntry = json.RawMessage

func (c *Client) FetchPortfolio(address string) (*PortfolioResponse, error) {
	payload, _ := json.Marshal(PortfolioRequest{
		Type: "portfolio",
		User: address,
	})

	resp, err := c.http.Post(infoURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("fetch portfolio for %s: %w", address, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read portfolio body: %w", err)
	}

	var result PortfolioResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal portfolio for %s: %w", address, err)
	}
	return &result, nil
}

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

// FetchUserFillsByTime 按时间范围获取用户成交记录
func (c *Client) FetchUserFillsByTime(address string, startTimeMs, endTimeMs int64) ([]Fill, error) {
	payload, _ := json.Marshal(FillsByTimeRequest{
		Type:      "userFillsByTime",
		User:      address,
		StartTime: startTimeMs,
		EndTime:   endTimeMs,
	})

	resp, err := c.http.Post(infoURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("fetch fills for %s: %w", address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429)")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read fills body: %w", err)
	}

	var fills []Fill
	if err := json.Unmarshal(body, &fills); err != nil {
		return nil, fmt.Errorf("unmarshal fills for %s: %w", address, err)
	}
	return fills, nil
}

// IsAtLimit 判断返回结果是否达到 API 上限
func IsAtLimit(fills []Fill) bool {
	return len(fills) >= fillsLimit
}
