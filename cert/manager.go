package cert

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"
)

// Manager 负责证书的获取、缓存、更新和降级策略。
type Manager struct {
	cfg    Config
	logger *zerolog.Logger

	// 内部状态
	manualCert  atomic.Pointer[tls.Certificate]
	acmeManager *autocert.Manager

	// 状态位：0=使用手动证书, 1=使用 ACME
	useACME atomic.Bool

	// 确保 Start 只执行一次
	startOnce sync.Once
}

// New 创建证书管理器。
func New(cfg Config, logger *zerolog.Logger) (*Manager, error) {
	m := &Manager{
		cfg:    cfg,
		logger: logger,
	}

	// 1. 初始化 ACME (如果启用)
	if cfg.ACME.Enabled {
		m.initACME()
	}

	// 2. 尝试初始加载手动证书
	if err := m.reloadFileCert(); err != nil {
		m.logger.Warn().Err(err).Msg("Failed to load manual certificate on startup")
		if cfg.ACME.Enabled {
			m.logger.Info().Msg("Falling back to ACME immediately")
			m.useACME.Store(true)
		}
	}

	return m, nil
}

// Start 启动后台监听（Watcher）。
func (m *Manager) Start(ctx context.Context) error {
	m.startOnce.Do(func() {
		// 只有配置了文件路径才启动文件监听
		if m.cfg.CertFile != "" && m.cfg.KeyFile != "" {
			go m.watchFileChanges(ctx)
		}
	})
	return nil
}

// Stop 停止管理器
func (m *Manager) Stop(ctx context.Context) error {
	return nil
}

// GetCertificate 实现 tls.Config.GetCertificate
// 这是一个高频调用的热点路径，实现了基于 atomic.Pointer 的无锁化读取。
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// 1. 优先检查是否启用了 ACME
	if m.useACME.Load() {
		if m.acmeManager != nil {
			return m.acmeManager.GetCertificate(hello)
		}
		m.logger.Warn().Msg("acme manager not init, falling back to manual certificate")
	}

	// 2. 否则使用手动加载的证书 (Lock-free Atomic Load)
	cert := m.manualCert.Load()

	// 3. 双重保险：如果手动证书不可用，尝试降级到 ACME
	if cert == nil {
		if m.acmeManager != nil {
			return m.acmeManager.GetCertificate(hello)
		}
		return nil, fmt.Errorf("cert manager: no certificate available for %s", hello.ServerName)
	}

	return cert, nil
}

// HTTPHandler ACME 挑战处理器
func (m *Manager) HTTPHandler(fallback http.Handler) http.Handler {
	if m.acmeManager != nil {
		return m.acmeManager.HTTPHandler(fallback)
	}
	return fallback
}
