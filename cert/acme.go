package cert

import (
	"golang.org/x/crypto/acme/autocert"
)

func (m *Manager) initACME() {
	cacheDir := m.cfg.ACME.CacheDir
	if cacheDir == "" {
		cacheDir = "./certs-cache"
	}

	hostPolicy := autocert.HostWhitelist(m.cfg.ACME.Domains...)
	if len(m.cfg.ACME.Domains) == 0 {
		// 如果未指定域名，允许所有（注意安全风险，通常建议指定）
		hostPolicy = nil
	}

	m.acmeManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Cache:      autocert.DirCache(cacheDir),
		Email:      m.cfg.ACME.Email,
	}
}
