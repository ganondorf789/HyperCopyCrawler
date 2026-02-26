package hyperliquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	leaderboardURL = "https://stats-data.hyperliquid.xyz/Mainnet/leaderboard"
	infoURL        = "https://api.hyperliquid.xyz/info"
)

// Client Hyperliquid API 客户端
type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 60 * time.Second},
	}
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

// TimeSeriesEntry 是 {"time": "...", "value": "..."} 或类似结构
// 实际返回的是按 window 分组的数据
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
