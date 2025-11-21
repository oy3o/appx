package appx

import (
	"time"

	"github.com/oy3o/appx/security"
	"github.com/rs/zerolog"
)

type Option func(*Appx)

// WithLogger 设置自定义 Logger
func WithLogger(l *zerolog.Logger) Option {
	return func(x *Appx) {
		x.logger = l
	}
}

// WithShutdownTimeout 设置优雅关闭的超时时间
func WithShutdownTimeout(d time.Duration) Option {
	return func(x *Appx) {
		x.shutdownTimeout = d
	}
}

// WithSecurityManager 注入安全检查管理器
func WithSecurityManager(mgr *security.Manager) Option {
	return func(x *Appx) {
		x.secMgr = mgr
	}
}

// WithConfig 注入配置对象，Appx 启动时会打印脱敏后的配置快照
func WithConfig(cfg any) Option {
	return func(x *Appx) {
		x.config = cfg
	}
}
