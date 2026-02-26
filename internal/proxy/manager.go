package proxy

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hypercopy/crawler/internal/hyperliquid"
	"github.com/hypercopy/crawler/internal/model"
	"gorm.io/gorm"
)

// Manager 代理池管理器，从数据库加载代理并轮询分配
type Manager struct {
	mu      sync.RWMutex
	proxies []model.ProxyPool
	index   atomic.Uint64
}

// NewManager 从数据库加载启用的代理
func NewManager(db *gorm.DB) (*Manager, error) {
	var proxies []model.ProxyPool
	if err := db.Where("status = ?", 1).Find(&proxies).Error; err != nil {
		return nil, fmt.Errorf("load proxies: %w", err)
	}
	log.Printf("[proxy] loaded %d active proxies", len(proxies))
	return &Manager{proxies: proxies}, nil
}

// Count 返回可用代理数量
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.proxies)
}

// GetByIndex 根据 worker 索引获取固定代理
func (m *Manager) GetByIndex(workerIdx int) *model.ProxyPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.proxies) == 0 {
		return nil
	}
	return &m.proxies[workerIdx%len(m.proxies)]
}

// Next 轮询获取下一个代理
func (m *Manager) Next() *model.ProxyPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.proxies) == 0 {
		return nil
	}
	idx := m.index.Add(1) - 1
	return &m.proxies[idx%uint64(len(m.proxies))]
}

// ProxyURL 构造代理 URL
func ProxyURL(p *model.ProxyPool) string {
	if p.Username != "" {
		return fmt.Sprintf("http://%s:%s@%s:%s", p.Username, p.Password, p.Host, p.Port)
	}
	return fmt.Sprintf("http://%s:%s", p.Host, p.Port)
}

// NewClientForWorker 为指定 worker 创建带代理的 API 客户端
func (m *Manager) NewClientForWorker(workerIdx int) (*hyperliquid.Client, error) {
	p := m.GetByIndex(workerIdx)
	if p == nil {
		// 无代理，返回直连客户端
		return hyperliquid.NewClient(), nil
	}
	return hyperliquid.NewClientWithProxy(ProxyURL(p))
}
