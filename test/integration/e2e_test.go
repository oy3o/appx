package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/oy3o/appx"
	"github.com/oy3o/httpx"
	"github.com/oy3o/o11y"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LogBuffer 是一个线程安全的 Buffer，用于捕获日志输出
type LogBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (l *LogBuffer) Write(p []byte) (n int, err error) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.b.Write(p)
}

func (l *LogBuffer) String() string {
	l.m.Lock()
	defer l.m.Unlock()
	return l.b.String()
}

// getFreePort 获取一个空闲的端口号
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// TestReq 模拟请求结构体
type TestReq struct {
	Message string `json:"message"`
}

// TestRes 模拟响应结构体
type TestRes struct {
	Reply string `json:"reply"`
}

// TestE2E_FullFlow 执行端到端全链路测试
// 验证：Appx -> Netx -> O11y(Trace) -> Httpx -> Log -> Response
func TestE2E_FullFlow(t *testing.T) {
	// 1. 准备环境
	port, err := getFreePort()
	require.NoError(t, err)
	addr := fmt.Sprintf("localhost:%d", port)
	baseUrl := fmt.Sprintf("http://%s", addr)

	// 捕获日志
	logBuf := &LogBuffer{}
	// 设置 zerolog 全局输出到 buffer (JSON格式方便解析)
	log.Logger = zerolog.New(logBuf).With().Timestamp().Logger()

	// 2. 初始化配置 (模拟 main.go)
	o11yCfg := o11y.Config{
		Enabled: true,
		Service: "e2e-test",
		Log: o11y.LogConfig{
			Level:         "info",
			EnableConsole: false, // 必须关闭 Console Writer，否则输出包含颜色代码且不是纯 JSON
			EnableFile:    false, // 我们手动接管了 Logger，这里设为 false
		},
		Trace: o11y.TraceConfig{
			Enabled:     true,
			Exporter:    "none", // 测试环境不发 OTLP
			SampleRatio: 1.0,
		},
		Metric: o11y.MetricConfig{
			Enabled:  false, // 简化测试，暂不验证 Metric
			Exporter: "none",
		},
	}

	// 初始化 o11y
	shutdownO11y := o11y.Init(o11yCfg)
	defer shutdownO11y(context.Background())

	// [关键] 注入 TraceID 获取器 (Task 3 的成果)
	httpx.GetTraceID = o11y.GetTraceID

	// 重新绑定 Logger 到 buffer (因为 o11y.Init 可能会重置 Logger)
	log.Logger = zerolog.New(logBuf).With().Timestamp().Logger()

	// 3. 构建 Appx 和 Handler
	mux := http.NewServeMux()
	// 注册一个测试路由
	mux.Handle("POST /echo", httpx.NewHandler(func(ctx context.Context, req *TestReq) (*TestRes, error) {
		// 在 Handler 内部打印日志，验证 TraceID 是否注入到了 Context 的 Logger 中
		logger := o11y.GetLoggerFromContext(ctx)
		logger.Info().Str("msg", req.Message).Msg("Handler executed")

		return &TestRes{Reply: "echo: " + req.Message}, nil
	}))

	// 4. 启动 Appx
	app := appx.New(
		appx.WithLogger(&log.Logger),
		appx.WithShutdownTimeout(2*time.Second),
	)

	httpSvc := appx.NewHttpService("e2e-api", addr, mux)
	// [关键] 启用自动化可观测性 (Task 2 的成果)
	httpSvc.WithObservability(o11yCfg)

	app.Add(httpSvc)

	// 在 goroutine 中运行 Appx，因为 Run 是阻塞的
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	// 5. 等待 Appx 就绪 (轮询健康检查或直接尝试连接)
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Appx failed to start within timeout")

	// 6. 发起 HTTP 请求
	reqBody := `{"message": "hello world"}`
	resp, err := http.Post(baseUrl+"/echo", "application/json", bytes.NewBufferString(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// 7. 验证响应 (Httpx)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证 Header 中包含 TraceID (Task 3)
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should be present")

	// 验证 JSON Body 中包含 TraceID
	var respBody httpx.Response[TestRes]
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)
	assert.Equal(t, "OK", respBody.Code)
	assert.Equal(t, "echo: hello world", respBody.Data.Reply)
	assert.Equal(t, traceIDHeader, respBody.TraceID, "Body TraceID should match Header")

	// 8. 验证日志 (Log Correlation)
	// 等待日志刷入 Buffer
	time.Sleep(100 * time.Millisecond)
	logOutput := logBuf.String()

	// 验证日志中包含 Handler 打印的消息
	assert.Contains(t, logOutput, "Handler executed")
	assert.Contains(t, logOutput, "hello world")

	// 验证日志中包含 TraceID 字段
	// zerolog 的 Context Logger 应该自动注入了 trace_id
	// 简单的字符串匹配验证
	assert.Contains(t, logOutput, fmt.Sprintf(`"trace_id":"%s"`, traceIDHeader), "Log should contain the TraceID")

	// 9. 优雅关闭
	// 发送 SIGTERM 信号给当前进程，Appx.Run 会捕获它
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	p.Signal(syscall.SIGTERM)

	// 等待 Appx 退出
	select {
	case err := <-errChan:
		assert.NoError(t, err, "Appx should exit gracefully")
	case <-time.After(3 * time.Second):
		t.Fatal("Appx failed to shutdown in time")
	}
}
