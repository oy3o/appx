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
		m.logger.Warn().Msg("ACME Domains are empty. HostPolicy will deny all requests. Please specify domains in config.")
	}

	m.acmeManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Cache:      autocert.DirCache(cacheDir),
		Email:      m.cfg.ACME.Email,
	}
}
