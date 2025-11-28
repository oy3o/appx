package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityFatal
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarn:
		return "WARN"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Result 封装检查结果
type Result struct {
	Name     string
	Passed   bool
	Severity Severity
	Message  string
	Error    error
}

// Checker 检查器接口
type Checker interface {
	Name() string
	Check(ctx context.Context) Result
}

// Manager 管理安全自检流程
type Manager struct {
	logger   *zerolog.Logger
	checkers []Checker
}

func New(logger *zerolog.Logger) *Manager {
	return &Manager{
		logger:   logger,
		checkers: make([]Checker, 0),
	}
}

// Register 注册检查项
func (m *Manager) Register(c ...Checker) {
	m.checkers = append(m.checkers, c...)
}

// Run 执行所有检查。
// 如果有 SeverityFatal 级别的检查失败，返回 error。
func (m *Manager) Run(ctx context.Context) error {
	m.logger.Info().Msg("Running security self-checks...")

	// 设置总超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	var fatalCount int
	var warnCount int

	for _, check := range m.checkers {
		c := check
		g.Go(func() error {
			// 捕获 Panic，防止单个 Checker 崩溃导致整个检查挂掉
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error().Str("checker", c.Name()).Interface("panic", r).Msg("Security checker panicked")
					// Panic 视为 Fatal 错误
					mu.Lock()
					fatalCount++
					mu.Unlock()
				}
			}()

			res := c.Check(ctx)

			if res.Passed {
				m.logger.Debug().Str("check", res.Name).Msg("Security check passed")
				return nil
			}

			// 记录结果
			msg := fmt.Sprintf("[%s] Check Failed: %s", res.Name, res.Message)

			mu.Lock()
			defer mu.Unlock()

			switch res.Severity {
			case SeverityInfo:
				m.logger.Info().Err(res.Error).Msg(msg)
			case SeverityWarn:
				warnCount++
				m.logger.Warn().Err(res.Error).Msg(msg)
			case SeverityFatal:
				fatalCount++
				m.logger.Error().Err(res.Error).Msg(msg)
			}
			return nil
		})
	}

	// 等待所有检查完成
	if err := g.Wait(); err != nil {
		return err
	}

	m.logger.Info().
		Int("fatal", fatalCount).
		Int("warn", warnCount).
		Msg("Security checks completed")

	if fatalCount > 0 {
		return fmt.Errorf("security check failed: %d fatal errors found", fatalCount)
	}

	return nil
}
