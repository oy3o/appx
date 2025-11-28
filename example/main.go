package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/oy3o/appx"
	"github.com/oy3o/appx/cert"
	"github.com/oy3o/appx/security"
	"github.com/oy3o/conf"
	"github.com/oy3o/httpx"
	"github.com/oy3o/o11y"
	"github.com/oy3o/task"
	"github.com/rs/zerolog/log"
)

// --- 1. 定义配置结构 ---

type Config struct {
	App struct {
		Name string `mapstructure:"name"`
		Addr string `mapstructure:"addr"`
	} `mapstructure:"app"`

	Monitor struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"monitor"`

	// 证书配置 (映射 cert.Config)
	Cert cert.Config `mapstructure:"cert"`

	// 可观测性配置
	O11y o11y.Config `mapstructure:"o11y"`
}

// --- 2. 定义业务请求/响应 (httpx) ---

type GreetReq struct {
	Name string `json:"name" validate:"required,min=2"`
	Age  int    `json:"age" validate:"gte=0"`
}

type GreetRes struct {
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

// --- 3. 定义业务逻辑 ---

func GreetHandler(ctx context.Context, req *GreetReq) (*GreetRes, error) {
	// 使用 o11y.GetLoggerFromContext 获取带 TraceID 的 Logger
	logger := o11y.GetLoggerFromContext(ctx)
	logger.Info().Str("user", req.Name).Msg("Processing greet request")

	if req.Name == "admin" {
		return nil, httpx.NewError(403, "FORBIDDEN", "admin is not allowed")
	}

	return &GreetRes{
		Message: fmt.Sprintf("Hello, %s! You are %d years old.", req.Name, req.Age),
		TraceID: o11y.GetTraceID(ctx),
	}, nil
}

func AsyncJobHandler(runner *task.Runner) httpx.HandlerFunc[GreetReq, string] {
	return func(ctx context.Context, req *GreetReq) (string, error) {
		// 提交到 Task Runner
		// 使用 o11y.GetLoggerFromContext 获取带 TraceID 的 Logger
		logger := o11y.GetLoggerFromContext(ctx)

		// 闭包传递拥有上下文的logger给后台任务
		err := runner.Submit(func(ctx context.Context) {
			lg := logger.With().Str("task", "email_sender").Logger()
			logger = &lg
			logger.Info().Str("to", req.Name).Msg("Sending email...")
			// 模拟耗时
			time.Sleep(500 * time.Millisecond)
			logger.Info().Msg("Email sent successfully")
		})
		if err != nil {
			if err == task.ErrQueueFull {
				return "", httpx.ErrTooManyRequests
			}
			return "", err
		}

		return "Task submitted successfully", nil
	}
}

// 示例 HealthChecker
type dbHealthChecker struct {
	db *sql.DB
}

func (c *dbHealthChecker) Name() string { return "database" }
func (c *dbHealthChecker) Check(ctx context.Context) error {
	if c.db == nil {
		return nil // 示例中没有真实 DB，直接返回成功
	}
	return c.db.PingContext(ctx)
}

// --- Main ---

func main() {
	// 1. 加载配置
	cfg, err := conf.Load[Config]("config", conf.WithSearchPaths("./example", "."))
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// 2. 初始化可观测性
	shutdownO11y, _ := o11y.Init(cfg.O11y)
	defer shutdownO11y(context.Background())

	// 3. 初始化基础组件

	// 3.1 证书管理器
	var certMgr *cert.Manager
	if cfg.Cert.CertFile != "" || cfg.Cert.ACME.Enabled {
		certMgr, err = cert.New(cfg.Cert, &log.Logger)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize cert manager")
		}
	}

	// 3.2 安全管理器
	secMgr := security.New(&log.Logger)
	secMgr.Register(
		&security.RootUserChecker{Severity: security.SeverityWarn},
		&security.BindAddrChecker{Addr: cfg.App.Addr, AllowPublic: true},
		&security.UlimitChecker{MinLimit: 4096, Severity: security.SeverityWarn},
	)

	// 3.3 任务执行器
	runner := task.NewRunner(task.WithMaxWorkers(10), task.WithQueueSize(100))

	// 4. 构建 Appx 容器
	app := appx.New(
		appx.WithLogger(&log.Logger),
		appx.WithSecurityManager(secMgr),
		appx.WithShutdownTimeout(10*time.Second),
		appx.WithConfig(cfg), // 启用配置快照打印
	)

	// 注册 Hooks
	app.AddShutdownHook(func(ctx context.Context) error {
		log.Info().Msg("Cleaning up resources (DB, Redis)...")
		return nil
	})

	// 注册 Health Checkers
	app.AddHealthChecker(&dbHealthChecker{db: nil})

	// 5. 注册服务

	// 5.1 Task Service
	app.Add(appx.NewTaskService(runner))

	// 5.2 Monitor Service (:9090)
	monitorAuth := func(ctx context.Context, user, pass string) (any, error) {
		if user == "admin" && pass == "s3cret" {
			return "admin", nil
		}
		return nil, fmt.Errorf("invalid credentials")
	}
	app.Add(appx.NewMonitorService(cfg.Monitor.Addr, app.HealthHandler(), httpx.AuthBasic(monitorAuth, "Monitor Area")))

	// 5.3 Main HTTP Service
	mux := http.NewServeMux()
	mux.Handle("POST /greet", httpx.NewHandler(GreetHandler))
	mux.Handle("POST /async", httpx.NewHandler(AsyncJobHandler(runner)))

	// 构建中间件链
	var httpHandler http.Handler = httpx.Chain(mux,
		httpx.DefaultCORS(),
	)

	// 应用 ACME HTTP-01 Challenge 中间件
	if certMgr != nil {
		httpHandler = certMgr.HTTPHandler(httpHandler)
	}

	// 创建服务
	httpSvc := appx.NewHttpService("main-api", cfg.App.Addr, httpHandler)
	httpSvc.WithLogger(&log.Logger)

	// 注入网络层中间件 (例如：限制 IP)
	// httpSvc.WithNetMiddleware(netx.WithAllowCIDR("192.168.0.0/16"))

	// 调整 TCP 参数
	httpSvc.WithKeepAlive(1 * time.Minute)

	// [Zero Config Update] 启用自动化可观测性
	httpSvc.WithObservability(cfg.O11y)

	// 绑定 TLS (如果已初始化)
	if certMgr != nil {
		httpSvc.WithTLS(certMgr)
	}

	// 设置连接保护参数
	httpSvc.WithMaxConns(5000)

	app.Add(httpSvc)

	// 6. 启动运行
	log.Info().Str("addr", cfg.App.Addr).Msg("Starting app...")
	if err := app.Run(); err != nil {
		log.Fatal().Err(err).Msg("Appx startup failed")
	}
}
