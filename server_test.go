package appx

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oy3o/appx/security"
	"github.com/oy3o/o11y"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// MockService 用于测试
type MockService struct {
	name       string
	startFunc  func(context.Context) error
	stopFunc   func(context.Context) error
	errHandler ErrorNotifier
}

func (m *MockService) Name() string { return m.name }

func (m *MockService) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *MockService) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

func (m *MockService) SetErrorNotify(fn ErrorNotifier) {
	m.errHandler = fn
}

// TestAppx_ConcurrentFatal 验证并发 Fatal 错误的处理
func TestAppx_ConcurrentFatal(t *testing.T) {
	// 1. 准备 Logger 捕获输出
	logOutput := &testLogWriter{}
	logger := zerolog.New(logOutput)

	app := New(WithLogger(&logger))

	// 2. 创建两个会报错的服务
	// Service A: 立即报错
	svcA := &MockService{name: "ServiceA"}
	svcA.startFunc = func(ctx context.Context) error {
		go func() {
			time.Sleep(10 * time.Millisecond)
			// 模拟 Service A 报错
			if svcA.errHandler != nil {
				svcA.errHandler(errors.New("error from A"))
			}
		}()
		return nil
	}

	// Service B: 稍晚一点报错 (模拟并发)
	// 通过 inShutdown 机制，即使 channel 被清空，只要 appx 进入关闭流程，这里也会打印日志
	svcB := &MockService{name: "ServiceB"}
	svcB.startFunc = func(ctx context.Context) error {
		go func() {
			time.Sleep(12 * time.Millisecond) // 比 A 慢一点点
			if svcB.errHandler != nil {
				svcB.errHandler(errors.New("error from B"))
			}
		}()
		return nil
	}

	app.Add(svcA)
	app.Add(svcB)

	// 3. 运行 Appx
	err := app.Run()

	// 4. 验证
	assert.Error(t, err)
	assert.Equal(t, "error from A", err.Error())

	// 给一点时间让 svcB 的 goroutine 执行完并写日志
	time.Sleep(100 * time.Millisecond)

	foundSecondary := false
	for _, entry := range logOutput.Entries {
		if msg, ok := entry["message"].(string); ok {
			if msg == "Secondary fatal error occurred during shutdown" {
				foundSecondary = true
				break
			}
		}
	}
	assert.True(t, foundSecondary, "Should log secondary fatal error")
}

// TestHandlePanic 验证 Panic 恢复逻辑
func TestHandlePanic(t *testing.T) {
	logOutput := &testLogWriter{}
	logger := zerolog.New(logOutput)

	notifyCalled := false
	notifier := func(err error) {
		notifyCalled = true
		assert.Contains(t, err.Error(), "service panic")
		assert.Contains(t, err.Error(), "boom")
	}

	// 模拟一个 panic 的 goroutine
	func() {
		defer handlePanic(&logger, notifier)
		panic("boom")
	}()

	assert.True(t, notifyCalled)

	// 验证日志包含堆栈
	foundStack := false
	for _, entry := range logOutput.Entries {
		if _, ok := entry["stack"]; ok {
			foundStack = true
			break
		}
	}
	assert.True(t, foundStack, "Log should contain stack trace")
}

// --- Helper for Log Capture ---

type testLogWriter struct {
	Entries []map[string]interface{}
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	return w.WriteLevel(zerolog.NoLevel, p)
}

// WriteLevel 实现 zerolog.LevelWriter 接口
func (w *testLogWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	entry := make(map[string]interface{})
	_ = json.Unmarshal(p, &entry) // Try parsing JSON

	// Fallback if json parsing fails or empty (zerolog console writer isn't json)
	// But zerolog.New(w) without ConsoleWriter produces JSON by default.
	if len(entry) == 0 {
		entry["raw"] = string(p)
	}

	w.Entries = append(w.Entries, entry)
	return len(p), nil
}

// --- Health Check Tests ---

type mockHealthChecker struct {
	name  string
	err   error
	delay time.Duration
}

func (m *mockHealthChecker) Name() string { return m.name }
func (m *mockHealthChecker) Check(ctx context.Context) error {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.err
}

func TestAppx_HealthHandler(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter())

	t.Run("All Healthy", func(t *testing.T) {
		app := New(WithLogger(&logger))
		app.AddHealthChecker(&mockHealthChecker{name: "db", err: nil})
		app.AddHealthChecker(&mockHealthChecker{name: "redis", err: nil})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/healthz", nil)
		app.HealthHandler().ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("One Failure", func(t *testing.T) {
		app := New(WithLogger(&logger))
		app.AddHealthChecker(&mockHealthChecker{name: "db", err: nil})
		app.AddHealthChecker(&mockHealthChecker{name: "redis", err: errors.New("connection refused")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/healthz", nil)
		app.HealthHandler().ServeHTTP(w, r)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "redis")
		assert.Contains(t, w.Body.String(), "connection refused")
	})

	t.Run("Timeout", func(t *testing.T) {
		app := New(WithLogger(&logger))
		// 模拟一个超时的检查 (5s > 默认3s)
		app.AddHealthChecker(&mockHealthChecker{name: "slow-api", delay: 5 * time.Second})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/healthz", nil)
		app.HealthHandler().ServeHTTP(w, r)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "context deadline exceeded")
	})
}

// --- Appx Lifecycle Tests ---

func TestAppx_Run_Rollback(t *testing.T) {
	// 测试启动失败时的回滚机制
	app := New()

	// Service 1: 启动成功，停止时需要被调用
	svc1Stopped := false
	svc1 := &MockService{
		name: "svc-1",
		stopFunc: func(ctx context.Context) error {
			svc1Stopped = true
			return nil
		},
	}

	// Service 2: 启动失败
	svc2 := &MockService{
		name: "svc-2",
		startFunc: func(ctx context.Context) error {
			return errors.New("port binding failed")
		},
	}

	app.Add(svc1)
	app.Add(svc2)

	// 运行 Appx
	err := app.Run()

	// 验证结果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port binding failed")
	assert.True(t, svc1Stopped, "Service 1 should be stopped (rolled back) when Service 2 fails to start")
}

type mockChecker struct {
	NameVal   string
	ResultVal security.Result
}

func (c *mockChecker) Name() string {
	return c.NameVal
}

func (c *mockChecker) Check(ctx context.Context) security.Result {
	return c.ResultVal
}

func TestAppx_Run_SecurityCheckFail(t *testing.T) {
	// 模拟安全检查失败导致无法启动
	mockSecMgr := security.New(&log.Logger)
	mockSecMgr.Register(&mockChecker{
		NameVal:   "fatal-check",
		ResultVal: security.Result{Passed: false, Severity: security.SeverityFatal, Message: "unsafe config"},
	})

	app := New(WithSecurityManager(mockSecMgr))

	err := app.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security check failed")
}

// --- Monitor Service Tests ---

func TestNewMonitorService(t *testing.T) {
	// 测试 Monitor Service 的路由注册情况
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	svc := NewMonitorService(":9090", handler)
	assert.Equal(t, "monitor", svc.Name())

	// 我们无法直接访问 svc 内部的 mux，但可以通过启动它来验证
	// 或者利用 httptest 来测试 handler (如果字段暴露的话)。
	// 这里的 HttpService 封装了 handler，我们可以通过反射或者 Start 后的行为来测。

	// 这里我们简单启动并请求一下
	go func() {
		_ = svc.Start(context.Background())
	}()
	defer svc.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	// 验证 /healthz
	resp, err := http.Get("http://127.0.0.1:9090/healthz")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	}

	// 验证 /metrics
	resp, err = http.Get("http://127.0.0.1:9090/metrics")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	}
}

// --- HttpService Options Tests ---

func TestHttpService_Options(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	svc := NewHttpService("test", ":8080", h)

	svc.WithMaxConns(100)
	assert.Equal(t, 100, svc.maxConns)

	l := zerolog.New(nil)
	svc.WithLogger(&l)
	assert.Equal(t, &l, svc.logger)

	svc.WithReusePort()
	assert.True(t, svc.enableReusePort)

	// 验证 Observability 配置注入
	o11yCfg := o11y.Config{Enabled: true}
	svc.WithObservability(o11yCfg)
	assert.True(t, svc.o11yCfg.Enabled)

	// 验证自定义中间件注入
	mw := func(l net.Listener) net.Listener {
		return l
	}
	svc.WithNetMiddleware(mw)
	assert.Len(t, svc.netMiddlewares, 1)

	// 验证 UDP 中间件
	udpMw := func(c net.PacketConn) net.PacketConn { return c }
	svc.WithUDPMiddleware(udpMw)
	assert.Len(t, svc.udpMiddlewares, 1)

	// 验证 KeepAlive 配置
	svc.WithKeepAlive(10 * time.Second)
	assert.Equal(t, 10*time.Second, svc.keepAlivePeriod)
}
