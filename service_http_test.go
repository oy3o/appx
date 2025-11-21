package appx

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oy3o/appx/cert"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTempCert 辅助生成测试用的自签名证书
func generateTempCert(t *testing.T) (certPath, keyPath string) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test Co"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	certPath = filepath.Join(tmpDir, "cert.pem")
	keyPath = filepath.Join(tmpDir, "key.pem")

	certOut, _ := os.Create(certPath)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(keyPath)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return
}

func TestHttpService_ConfigValidation(t *testing.T) {
	// 场景：开启 HTTP/3 但未开启 TLS -> 应该报错
	svc := NewHttpService("test", ":0", nil).WithHTTP3()
	err := svc.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP/3 requires TLS")
}

func TestHttpService_Integration_H3(t *testing.T) {
	// 1. 准备证书Logger
	cPath, kPath := generateTempCert(t)
	certMgr, err := cert.New(cert.Config{CertFile: cPath, KeyFile: kPath}, &log.Logger)
	require.NoError(t, err)

	// 2. 准备 Handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello h3"))
	})

	// 3. 获取随机端口
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	// 4. 创建服务 (开启 TLS, HTTP/3, ReusePort)
	// logger := zerolog.New(zerolog.NewConsoleWriter())
	svc := NewHttpService("h3-svc", addr, handler).
		WithTLS(certMgr).
		WithHTTP3().
		WithReusePort().
		WithLogger(&zerolog.Logger{})

	// 5. 异步启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- svc.Start(ctx)
	}()

	// 6. 等待启动就绪
	require.Eventually(t, func() bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Appx failed to start port")

	// 7. 验证 TCP/HTTPS 访问及 Alt-Svc 头
	// 创建信任自签名证书的客户端
	caCert, _ := os.ReadFile(cPath)
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caPool},
		},
	}

	// 发起 HTTPS 请求 (使用 TCP)
	resp, err := client.Get("https://" + addr)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "hello h3", string(body))

	// 关键验证：检查 HTTP/3 升级头
	// 期望类似于: h3=":port"; ma=2592000
	altSvc := resp.Header.Get("Alt-Svc")
	fmt.Printf("Got Alt-Svc: %s\n", altSvc)
	assert.Contains(t, altSvc, "h3", "Response should contain Alt-Svc header advertising HTTP/3")

	// 8. 停止服务
	cancel()
	err = svc.Stop(context.Background())
	assert.NoError(t, err)

	// 检查 Start 是否有错误返回
	select {
	case err := <-errChan:
		// context cancel 导致的返回通常是 http.ErrServerClosed 或 nil (取决于实现细节)
		// 我们的 Start 实现里忽略了 ErrServerClosed
		assert.NoError(t, err)
	default:
	}
}
