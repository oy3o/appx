package security

import (
	"context"
	"fmt"
	"os"
	"runtime"
)

// RootUserChecker 检查是否以 Root 身份运行
type RootUserChecker struct {
	Severity Severity
}

func (c *RootUserChecker) Name() string { return "root_user" }

func (c *RootUserChecker) Check(ctx context.Context) Result {
	if runtime.GOOS == "windows" {
		return Result{Name: c.Name(), Passed: true}
	}

	if os.Geteuid() == 0 {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  "Application is running as root (UID 0). This is insecure.",
		}
	}
	return Result{Name: c.Name(), Passed: true}
}

// FilePermChecker 检查关键文件权限 (如 0600)
type FilePermChecker struct {
	Path     string
	MaxPerm  os.FileMode // 例如 0600
	Severity Severity
}

func (c *FilePermChecker) Name() string { return fmt.Sprintf("file_perm:%s", c.Path) }

func (c *FilePermChecker) Check(ctx context.Context) Result {
	info, err := os.Stat(c.Path)
	if err != nil {
		// 如果文件必须存在，这应该是一个 Fatal 错误，但这里只返回检查结果
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  fmt.Sprintf("File not found or not readable: %s", c.Path),
			Error:    err,
		}
	}

	// 检查是否有超出 MaxPerm 的权限位被设置
	if info.Mode().Perm()&^c.MaxPerm != 0 {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  fmt.Sprintf("Insecure permissions: got %o, max allowed %o", info.Mode().Perm(), c.MaxPerm),
		}
	}

	return Result{Name: c.Name(), Passed: true}
}

// ConfigChecker 这是一个通用的配置检查器，传入一个闭包
type ConfigChecker struct {
	ID       string
	Severity Severity
	CheckFn  func() (bool, string)
}

func (c *ConfigChecker) Name() string { return "config:" + c.ID }

func (c *ConfigChecker) Check(ctx context.Context) Result {
	passed, msg := c.CheckFn()
	if !passed {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  msg,
		}
	}
	return Result{Name: c.Name(), Passed: true}
}
