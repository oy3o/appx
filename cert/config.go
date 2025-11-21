package cert

// ACME (Let's Encrypt) 配置
type ACME struct {
	Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
	Email    string   `mapstructure:"email" yaml:"email"`
	Domains  []string `mapstructure:"domains" yaml:"domains"`
	CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
}

type Config struct {
	// 手动证书路径
	CertFile string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" yaml:"key_file"`

	ACME ACME `mapstructure:"acme" yaml:"acme"`

	// 降级阈值：如果手动证书还有多少天过期，就切换到 ACME (默认 30 天)
	// 如果为 0，表示只有文件不存在或已完全过期才切换
	FallbackThresholdDays int `mapstructure:"fallback_threshold_days" yaml:"fallback_threshold_days"`
}

func DefaultConfig() Config {
	return Config{
		FallbackThresholdDays: 30,
	}
}
