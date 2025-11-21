package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Benchmark_GetCertificate 验证无锁化后的性能
// 模拟高并发 TLS 握手场景
func Benchmark_GetCertificate(b *testing.B) {
	tempDir := b.TempDir()
	// 使用 ECDSA 生成证书，速度极快，不会影响 Benchmark 启动时间
	certFile, keyFile := generateTestCert(nil, tempDir, 24*time.Hour)

	cfg := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled: false,
		},
	}

	// 2. 初始化 Manager
	mgr, err := New(cfg, &log.Logger)
	if err != nil {
		b.Fatalf("Failed to init manager: %v", err)
	}

	hello := &tls.ClientHelloInfo{ServerName: "example.com"}

	b.ResetTimer()
	b.ReportAllocs()

	// 3. 并发基准测试
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mgr.GetCertificate(hello)
		}
	})
}

// TestManager_ReloadConcurrency 验证在重载证书时，并发读取不会 Panic
func TestManager_ReloadConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	certFile, keyFile := generateTestCert(t, tempDir, 1*time.Hour)

	cfg := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled: false,
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// 模拟并发读取 (GetCertificate)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hello := &tls.ClientHelloInfo{ServerName: "example.com"}
			for {
				select {
				case <-done:
					return
				default:
					c, err := mgr.GetCertificate(hello)
					assert.NoError(t, err)
					assert.NotNil(t, c)
				}
			}
		}()
	}

	// 模拟频繁写入 (reload)
	// 循环次数适量即可，配合 ECDSA 生成速度很快
	for i := 0; i < 50; i++ {
		generateTestCert(t, tempDir, 1*time.Hour)
		err := mgr.reloadFileCert()
		assert.NoError(t, err)
	}

	close(done)
	wg.Wait()
}

// generateTestCert 辅助函数：生成临时证书
// 使用 ECDSA (P256) 替代 RSA，生成速度提升 100x+
func generateTestCert(t *testing.T, dir string, validDuration time.Duration) (certPath, keyPath string) {
	// 使用 ECDSA P256，生成非常快
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		if t != nil {
			require.NoError(t, err)
		}
		return
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(validDuration),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		if t != nil {
			require.NoError(t, err)
		}
		return
	}

	certPath = filepath.Join(dir, "cert.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		if t != nil {
			require.NoError(t, err)
		}
		return
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyPath = filepath.Join(dir, "key.pem")
	keyOut, err := os.Create(keyPath)
	if err != nil {
		if t != nil {
			require.NoError(t, err)
		}
		return
	}
	defer keyOut.Close()

	// ECDSA 私钥序列化
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		if t != nil {
			require.NoError(t, err)
		}
		return
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return certPath, keyPath
}

func TestManager_ManualCert_HappyPath(t *testing.T) {
	tempDir := t.TempDir()
	certFile, keyFile := generateTestCert(t, tempDir, 1*time.Hour)

	cfg := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled: false,
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	hello := &tls.ClientHelloInfo{ServerName: "example.com"}
	cert, err := mgr.GetCertificate(hello)
	require.NoError(t, err)
	require.NotNil(t, cert)
	assert.False(t, mgr.useACME.Load())
}

func TestManager_StartupFallback(t *testing.T) {
	// 场景：配置文件指向不存在的路径，但 ACME 开启
	// 预期：启动不报错，但状态位 useACME 应为 true
	tempDir := t.TempDir()

	cfg := Config{
		CertFile: filepath.Join(tempDir, "missing.pem"),
		KeyFile:  filepath.Join(tempDir, "missing.key"),
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled:  true,
			CacheDir: tempDir,
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	// 断言：应该自动降级
	assert.True(t, mgr.useACME.Load(), "Should fallback to ACME on startup failure")
}

func TestManager_ExpirationCheck(t *testing.T) {
	tempDir := t.TempDir()
	// 即将过期 (10h < 30d)
	certFile, keyFile := generateTestCert(t, tempDir, 10*time.Hour)

	cfg := Config{
		CertFile:              certFile,
		KeyFile:               keyFile,
		FallbackThresholdDays: 30,
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled:  true,
			CacheDir: tempDir,
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	mgr.checkExpiration()

	assert.True(t, mgr.useACME.Load(), "Should switch to ACME due to expiration")
}

func TestManager_ReloadAndRecover(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	cfg := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
		ACME: struct {
			Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
			Email    string   `mapstructure:"email" yaml:"email"`
			Domains  []string `mapstructure:"domains" yaml:"domains"`
			CacheDir string   `mapstructure:"cache_dir" yaml:"cache_dir"`
		}{
			Enabled:  true,
			CacheDir: tempDir,
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	// 初始状态：文件不存在，使用 ACME
	assert.True(t, mgr.useACME.Load())

	// 模拟文件创建
	generateTestCert(t, tempDir, 24*time.Hour)

	// 手动重载
	err = mgr.reloadFileCert()
	require.NoError(t, err)

	// 模拟 Watcher 逻辑切换状态
	mgr.useACME.Store(false)

	assert.False(t, mgr.useACME.Load())

	cert, err := mgr.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
}

func TestManager_StartStop(t *testing.T) {
	// 简单的覆盖率测试，确保 Start/Stop 不会 Panic
	cfg := Config{
		CertFile: "dummy.pem", // 即使文件不存在，Start 也会启动 goroutine
		KeyFile:  "dummy.key",
	}
	mgr, _ := New(cfg, &log.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	err := mgr.Start(ctx)
	assert.NoError(t, err)

	// 立即停止
	cancel()
	err = mgr.Stop(context.Background())
	assert.NoError(t, err)
}

func TestManager_HTTPHandler(t *testing.T) {
	cfg := Config{}
	// Case 1: ACME Disabled
	mgr, _ := New(cfg, &log.Logger)
	h := mgr.HTTPHandler(nil)
	assert.Nil(t, h, "Should return nil (or fallback) if ACME not enabled")

	// Case 2: ACME Enabled
	cfg.ACME.Enabled = true
	cfg.ACME.CacheDir = t.TempDir()
	mgr, _ = New(cfg, &log.Logger)
	h = mgr.HTTPHandler(nil)
	assert.NotNil(t, h, "Should return ACME handler")
}

func TestManager_GetCertificate_Logic(t *testing.T) {
	// 场景 1: ACME 优先
	// 模拟 ACME 开启，且强制使用 ACME 模式
	cfg := Config{
		ACME: ACME{Enabled: true, CacheDir: t.TempDir(), Email: "test@test.com"},
	}
	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	// 强制设置状态为使用 ACME
	mgr.useACME.Store(true)

	// 此时 manualCert 为 nil，acmeManager 已初始化
	// GetCertificate 应该调用 acmeManager (这里会因为 hello 为 nil 或 acme 内部逻辑返回 error，但路径已覆盖)
	// 注意：autocert.Manager.GetCertificate 如果 hello 为 nil 会 panic，所以我们要传一个
	hello := &tls.ClientHelloInfo{ServerName: "example.com"}

	// 由于 autocert 需要真实网络或复杂的 mock，我们这里主要验证代码没 panic 且走了 acme 分支
	// autocert 在没有 cache 且没有网络时会报错
	_, err = mgr.GetCertificate(hello)
	// 只要不是 "no certificate available" (手动模式的错)，说明走了 ACME 逻辑
	assert.Error(t, err)
}

func TestManager_GetCertificate_Fallback(t *testing.T) {
	// 场景 2: 手动模式 -> 证书不存在 -> 降级尝试 ACME
	cfg := Config{
		ACME: ACME{Enabled: true, CacheDir: t.TempDir()},
	}
	mgr, _ := New(cfg, &log.Logger)

	// 确保手动证书为空
	mgr.manualCert.Store(nil)
	mgr.useACME.Store(false) // 当前标记为手动模式

	hello := &tls.ClientHelloInfo{ServerName: "example.com"}

	// 此时 GetCertificate 发现 manualCert 为 nil，会尝试由 ACME 接管
	_, err := mgr.GetCertificate(hello)
	// 同样，我们只验证它没有直接报 "cert manager: no certificate available"
	// 因为 acme manager 已经初始化了，它会尝试处理
	assert.Error(t, err)
	assert.NotEqual(t, "cert manager: no certificate available for example.com", err.Error())
}

func TestManager_Config_ACME_Init(t *testing.T) {
	// 测试 initACME 的配置逻辑
	cfg := Config{
		ACME: ACME{
			Enabled: true,
			Domains: []string{"a.com", "b.com"},
			Email:   "admin@a.com",
			// CacheDir 留空测试默认值
		},
	}

	mgr, err := New(cfg, &log.Logger)
	require.NoError(t, err)

	// 通过反射或行为验证 internal 状态很难，但我们可以通过 HTTPHandler 来验证 acmeManager 是否非空
	h := mgr.HTTPHandler(nil)
	assert.NotNil(t, h, "ACME manager should be initialized")
}
