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

	"github.com/hypercopy/crawler/internal/model"
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

func (c *Client) FetchLeaderboard() (*model.LeaderboardResponse, error) {
	resp, err := c.http.Get(leaderboardURL)
	if err != nil {
		return nil, fmt.Errorf("fetch leaderboard: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read leaderboard body: %w", err)
	}

	var result model.LeaderboardResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal leaderboard: %w", err)
	}
	return &result, nil
}

// --- Portfolio (accountValueHistory + pnlHistory) ---

func (c *Client) FetchPortfolio(address string) (*model.PortfolioResponse, error) {
	payload, _ := json.Marshal(model.PortfolioRequest{
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

	var result model.PortfolioResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal portfolio for %s: %w", address, err)
	}
	return &result, nil
}

// --- UserFillsByTime ---

// FetchUserFillsByTime 按时间范围获取用户成交记录
func (c *Client) FetchUserFillsByTime(address string, startTimeMs, endTimeMs int64) ([]model.Fill, error) {
	payload, _ := json.Marshal(model.FillsByTimeRequest{
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

	var fills []model.Fill
	if err := json.Unmarshal(body, &fills); err != nil {
		return nil, fmt.Errorf("unmarshal fills for %s: %w", address, err)
	}
	return fills, nil
}

// IsAtLimit 判断返回结果是否达到 API 上限
func IsAtLimit(fills []model.Fill) bool {
	return len(fills) >= fillsLimit
}

// --- ClearinghouseState (永续合约持仓 + 保证金) ---

func (c *Client) FetchClearinghouseState(address string) (*model.ClearinghouseState, error) {
	payload, _ := json.Marshal(map[string]string{
		"type": "clearinghouseState",
		"user": address,
	})

	resp, err := c.http.Post(infoURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("fetch clearinghouse for %s: %w", address, err)
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
		return nil, fmt.Errorf("read clearinghouse body: %w", err)
	}

	var result model.ClearinghouseState
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal clearinghouse for %s: %w", address, err)
	}
	return &result, nil
}

// --- SpotClearinghouseState (现货持仓) ---

func (c *Client) FetchSpotClearinghouseState(address string) (*model.SpotClearinghouseState, error) {
	payload, _ := json.Marshal(map[string]string{
		"type": "spotClearinghouseState",
		"user": address,
	})

	resp, err := c.http.Post(infoURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("fetch spot clearinghouse for %s: %w", address, err)
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
		return nil, fmt.Errorf("read spot clearinghouse body: %w", err)
	}

	var result model.SpotClearinghouseState
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal spot clearinghouse for %s: %w", address, err)
	}
	return &result, nil
}
