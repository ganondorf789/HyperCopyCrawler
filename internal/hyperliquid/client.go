package hyperliquid

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hypercopy/crawler/internal/model"
	"go.uber.org/zap"
)

const (
	leaderboardURL = "https://stats-data.hyperliquid.xyz/Mainnet/leaderboard"
	infoURL        = "https://api.hyperliquid.xyz/info"

	fillsLimit   = 2000 // API 单次返回上限
	fundingLimit = 500  // 资金费 API 单次返回上限
	ordersLimit  = 2000 // 历史委托 API 单次返回上限
	maxRetries   = 3    // 429 限频最大重试次数
)

// ErrRateLimited 429 限频重试耗尽
var ErrRateLimited = errors.New("rate limited")

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

// postInfoWithRetry POST 请求 infoURL，遇到 429 自动重试（最多 maxRetries 次）
func (c *Client) postInfoWithRetry(payload []byte) ([]byte, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.http.Post(infoURL, "application/json", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			if attempt < maxRetries {
				wait := time.Duration(5*(attempt+1)) * time.Second
				zap.S().Warnf("[api] 429 rate limited, retry %d/%d after %v", attempt+1, maxRetries, wait)
				time.Sleep(wait)
				continue
			}
			zap.S().Warnf("[api] 429 rate limited, all %d retries exhausted", maxRetries)
			return nil, ErrRateLimited
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return body, nil
	}
	return nil, ErrRateLimited
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

func (c *Client) FetchPortfolio(address string) ([]model.PortfolioWindowEntry, error) {
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

	var result []model.PortfolioWindowEntry
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal portfolio for %s: %w", address, err)
	}
	return result, nil
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

	body, err := c.postInfoWithRetry(payload)
	if err != nil {
		return nil, fmt.Errorf("fetch fills for %s: %w", address, err)
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

// --- UserFundingHistory ---

// FetchUserFundingHistory 按时间范围获取用户资金费记录
func (c *Client) FetchUserFundingHistory(address string, startTimeMs, endTimeMs int64) ([]model.FundingEntry, error) {
	payload, _ := json.Marshal(model.FundingHistoryRequest{
		Type:      "userFunding",
		User:      address,
		StartTime: startTimeMs,
		EndTime:   endTimeMs,
	})

	body, err := c.postInfoWithRetry(payload)
	if err != nil {
		return nil, fmt.Errorf("fetch funding for %s: %w", address, err)
	}

	var entries []model.FundingEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal funding for %s: %w", address, err)
	}
	return entries, nil
}

// IsFundingAtLimit 判断资金费返回结果是否达到 API 上限
func IsFundingAtLimit(entries []model.FundingEntry) bool {
	return len(entries) >= fundingLimit
}

// --- HistoricalOrders ---

// FetchHistoricalOrders 按时间范围获取用户历史委托记录
func (c *Client) FetchHistoricalOrders(address string, startTimeMs, endTimeMs int64) ([]model.OrderEntry, error) {
	payload, _ := json.Marshal(model.HistoricalOrdersRequest{
		Type:      "historicalOrders",
		User:      address,
		StartTime: startTimeMs,
		EndTime:   endTimeMs,
	})

	body, err := c.postInfoWithRetry(payload)
	if err != nil {
		return nil, fmt.Errorf("fetch orders for %s: %w", address, err)
	}

	var orders []model.OrderEntry
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, fmt.Errorf("unmarshal orders for %s: %w", address, err)
	}
	return orders, nil
}

// IsOrdersAtLimit 判断历史委托返回结果是否达到 API 上限
func IsOrdersAtLimit(orders []model.OrderEntry) bool {
	return len(orders) >= ordersLimit
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
