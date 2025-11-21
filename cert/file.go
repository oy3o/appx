package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// watchFileChanges 定期检查证书文件状态
func (m *Manager) watchFileChanges(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// 初始化 lastMod，防止启动时如果文件存在但很快被修改导致第一次变更被忽略
	// 或者直接设为零值，第一次循环肯定会触发检查
	var lastMod time.Time
	if info, err := os.Stat(m.cfg.CertFile); err == nil {
		lastMod = info.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(m.cfg.CertFile)
			if err != nil {
				// 文件丢失
				if m.cfg.ACME.Enabled && !m.useACME.Load() {
					m.logger.Warn().Err(err).Msg("Certificate file missing, switching to ACME")
					m.useACME.Store(true)
				}
				continue
			}

			// 检查是否需要重载：从 ACME 恢复 或 文件被修改
			shouldReload := m.useACME.Load() || !info.ModTime().Equal(lastMod)

			if shouldReload {
				// 避免死循环：如果是恢复模式且文件没变（说明上次reload失败了），跳过
				if m.useACME.Load() && info.ModTime().Equal(lastMod) {
					continue
				}

				if err := m.reloadFileCert(); err != nil {
					m.logger.Error().Err(err).Msg("Failed to reload certificate")
				} else {
					// 加载成功
					lastMod = info.ModTime()
					if m.useACME.Load() {
						m.logger.Info().Msg("Certificate restored, switching back to manual mode")
						m.useACME.Store(false)
					}
				}
			}

			// 检查过期时间 (仅在手动模式下)
			if !m.useACME.Load() {
				m.checkExpiration()
			}
		}
	}
}

// reloadFileCert 从磁盘加载证书并解析
func (m *Manager) reloadFileCert() error {
	cert, err := tls.LoadX509KeyPair(m.cfg.CertFile, m.cfg.KeyFile)
	if err != nil {
		return err
	}

	if len(cert.Certificate) == 0 {
		return fmt.Errorf("no certificate found in %s", m.cfg.CertFile)
	}

	// 手动解析 Leaf 以便后续检查过期时间
	if len(cert.Certificate) > 0 {
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return err
		}
	}

	// 原子替换，无锁操作
	m.manualCert.Store(&cert)

	m.logger.Info().
		Str("file", m.cfg.CertFile).
		Time("expires", cert.Leaf.NotAfter).
		Msg("Certificate loaded from file")
	return nil
}

// checkExpiration 检查当前手动证书是否即将过期
func (m *Manager) checkExpiration() {
	// 原子读取
	cert := m.manualCert.Load()
	if cert == nil || cert.Leaf == nil {
		return
	}

	// 计算剩余时间
	timeLeft := time.Until(cert.Leaf.NotAfter)
	threshold := time.Duration(m.cfg.FallbackThresholdDays) * 24 * time.Hour

	// 如果剩余时间小于阈值，且启用了 ACME，且当前未在使用 ACME
	if timeLeft < threshold && m.cfg.ACME.Enabled && !m.useACME.Load() {
		m.logger.Warn().
			Dur("time_left", timeLeft).
			Dur("threshold", threshold).
			Msg("Manual certificate is expiring soon, switching to ACME fallback")
		m.useACME.Store(true)
	}
}
