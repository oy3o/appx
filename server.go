package appx

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/oy3o/appx/security"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Appx struct {
	config          any
	logger          *zerolog.Logger
	shutdownTimeout time.Duration
	secMgr          *security.Manager

	services       []Service
	hooks          []ShutdownHook
	healthCheckers []HealthChecker

	// fatalChan 用于接收 Service 运行时的致命错误
	fatalChan chan error
	// inShutdown 标记服务器是否已进入关闭流程
	inShutdown atomic.Bool
}

func New(opts ...Option) *Appx {
	s := &Appx{
		shutdownTimeout: 30 * time.Second,
		services:        make([]Service, 0),
		hooks:           make([]ShutdownHook, 0),
		healthCheckers:  make([]HealthChecker, 0),
		fatalChan:       make(chan error, 1),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.logger == nil {
		s.logger = &log.Logger
	}
	return s
}

// Add 注册服务
func (s *Appx) Add(svc Service) {
	if notifier, ok := svc.(ErrorNotifiable); ok {
		notifier.SetErrorNotify(s.notifyFatalError)
	}
	s.services = append(s.services, svc)
}

// AddShutdownHook 注册关闭钩子
func (s *Appx) AddShutdownHook(hook ShutdownHook) {
	s.hooks = append(s.hooks, hook)
}

// AddHealthChecker 注册健康检查
func (s *Appx) AddHealthChecker(checker HealthChecker) {
	s.healthCheckers = append(s.healthCheckers, checker)
}

// notifyFatalError 内部回调
func (s *Appx) notifyFatalError(err error) {
	// 如果已经开始关闭，直接记录日志，不再尝试发送通道
	if s.inShutdown.Load() {
		s.logger.Error().Err(err).Msg("Secondary fatal error occurred during shutdown")
		return
	}

	select {
	case s.fatalChan <- err:
		// 成功发送，触发关闭
	default:
		// Channel 已满，记录日志
		s.logger.Error().Err(err).Msg("Secondary fatal error occurred during shutdown")
	}
}

// handlePanic 是一个内部辅助函数，用于在 Service 的 goroutine 中捕获 Panic。
// 它会记录堆栈信息并通过 notifyFatalError 通知 Appx 关闭。
// 使用方法: defer handlePanic(logger, notifier)
func handlePanic(logger *zerolog.Logger, notifier ErrorNotifier) {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := fmt.Errorf("service panic: %v", r)

		// 1. 记录日志 (包含堆栈)
		if logger != nil {
			logger.Error().
				Interface("panic", r).
				Str("stack", string(stack)).
				Msg("Service crashed with panic")
		}

		// 2. 通知关闭
		if notifier != nil {
			notifier(err)
		}
	}
}

// HealthHandler 返回一个标准的 http.Handler 用于 /healthz
func (s *Appx) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 创建一个带有超时的上下文，防止整个健康检查请求耗时过长
		// 这里设置 3 秒作为总超时，你可以根据需求调整
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		// 2. 创建 errgroup
		g, ctx := errgroup.WithContext(ctx)

		// 3. 遍历所有检查器，并发执行
		for _, c := range s.healthCheckers {
			c := c // 捕获循环变量 (Go 1.22+ 不需要这行，但在旧版本是必须的)

			g.Go(func() error {
				// 为单个检查器设置独立的更短的超时（例如 2秒），
				// 这样可以确保某个特定组件慢不会占满整个 3秒 的总时间配额
				// (可选，或者直接使用外层的 ctx)
				checkCtx, checkCancel := context.WithTimeout(ctx, 2*time.Second)
				defer checkCancel()

				if err := c.Check(checkCtx); err != nil {
					return fmt.Errorf("[%s] %w", c.Name(), err)
				}
				return nil
			})
		}

		// 4. 等待结果
		// errgroup 会返回第一个出现的错误，且一旦有错误，ctx 会被 cancel，
		// 其他正在进行的检查如果监听了 ctx 也会尽快退出。
		if err := g.Wait(); err != nil {
			s.logger.Warn().Err(err).Msg("Health check failed")

			// 返回 503 和具体的错误信息
			http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

func (s *Appx) Run() error {
	// 0. 打印配置快照 (New Feature)
	if s.config != nil {
		printConfigSnapshot(s.logger, s.config)
	}

	// 1. 安全自检
	if s.secMgr != nil {
		if err := s.secMgr.Run(context.Background()); err != nil {
			s.logger.Error().Err(err).Msg("Security check failed")
			return err
		}
	}

	// 创建根 Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. 启动服务
	// 由于 Service.Start 实现约定为非阻塞（内部 go func），这里直接顺序启动即可。
	// 任何启动时的立即错误（如端口被占用）会立刻返回。
	var startedServices []Service // 记录已启动的服务

	for _, svc := range s.services {
		if err := svc.Start(ctx); err != nil {
			s.logger.Error().Err(err).Str("name", svc.Name()).Msg("Service failed to start, rolling back...")

			// 回滚：停止已启动的服务
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for i := len(startedServices) - 1; i >= 0; i-- {
				_ = startedServices[i].Stop(rollbackCtx)
			}

			return fmt.Errorf("service %s start failed: %w", svc.Name(), err)
		}
		startedServices = append(startedServices, svc)
	}

	// 3. 信号监听与错误捕获
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var shutdownReason string
	var returnErr error // 用于记录导致退出的错误

	select {
	// 由于 Start 是非阻塞的，ctx.Done() 只有在外部 cancel 时才会触发，或者配合其他 Context 管理
	// 这里主要依赖 fatalChan 和 quit
	case sig := <-quit:
		shutdownReason = fmt.Sprintf("signal received: %s", sig)
	case err := <-s.fatalChan:
		shutdownReason = fmt.Sprintf("fatal service error: %v", err)
		returnErr = err // 捕获错误用于返回
	}

	// 标记进入关闭状态
	s.inShutdown.Store(true)

	s.logger.Info().Str("reason", shutdownReason).Msg("Appx shutting down...")
	cancel()

	// 4. 优雅关闭流程
	s.logger.Info().Msg("Shutting down appx...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer shutdownCancel()

	// 4.1 倒序停止 Service (先停入口，再停后台)
	for i := len(s.services) - 1; i >= 0; i-- {
		svc := s.services[i]
		s.logger.Info().Str("name", svc.Name()).Msg("Stopping service")
		if err := svc.Stop(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Str("name", svc.Name()).Msg("Service stop error")
		}
	}

	// 4.2 执行 Shutdown Hooks (关闭 DB, Redis 等)
	for _, hook := range s.hooks {
		if err := hook(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Msg("Shutdown hook error")
		}
	}

	s.logger.Info().Msg("Appx stopped gracefully")
	return returnErr
}
