//go:build linux

package security

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// UlimitChecker 检查文件描述符限制 (FD Limit)
type UlimitChecker struct {
	MinLimit uint64
	Severity Severity
}

func (c *UlimitChecker) Name() string { return "os_ulimit" }

func (c *UlimitChecker) Check(ctx context.Context) Result {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return Result{
			Name: c.Name(), Passed: false, Severity: SeverityWarn,
			Error: err, Message: "Failed to get RLIMIT_NOFILE",
		}
	}

	if rLimit.Cur < c.MinLimit {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  fmt.Sprintf("Soft FD limit is too low: %d (recommended >= %d). May affect high concurrency.", rLimit.Cur, c.MinLimit),
		}
	}
	return Result{Name: c.Name(), Passed: true}
}

// SysctlChecker 检查内核参数 (/proc/sys)
type SysctlChecker struct {
	Key      string // e.g., "net.core.somaxconn"
	MinValue int64
	Severity Severity
}

func (c *SysctlChecker) Name() string { return "os_sysctl:" + c.Key }

func (c *SysctlChecker) Check(ctx context.Context) Result {
	// 将点号转换为路径，例如 net.core.somaxconn -> /proc/sys/net/core/somaxconn
	path := "/proc/sys/" + strings.ReplaceAll(c.Key, ".", "/")

	content, err := os.ReadFile(path)
	if err != nil {
		// 在某些容器环境（如无特权容器），/proc/sys 可能不可读
		// 降级为 Info，不报错
		return Result{
			Name:     c.Name(),
			Passed:   true, // 跳过视为通过
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("Skipped: cannot read sysctl %s", c.Key),
		}
	}

	valStr := strings.TrimSpace(string(content))
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return Result{Name: c.Name(), Passed: false, Severity: SeverityWarn, Error: err, Message: "Invalid sysctl value format"}
	}

	if val < c.MinValue {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  fmt.Sprintf("Kernel param %s is %d (recommended >= %d). Performance may be throttled.", c.Key, val, c.MinValue),
		}
	}

	return Result{Name: c.Name(), Passed: true}
}

// SwapChecker 检查系统是否开启了 Swap
// 对于 Go GC 来说，Swap 是性能杀手。生产环境建议关闭。
type SwapChecker struct {
	Severity Severity
}

func (c *SwapChecker) Name() string { return "os_swap" }

func (c *SwapChecker) Check(ctx context.Context) Result {
	f, err := os.Open("/proc/swaps")
	if err != nil {
		return Result{Name: c.Name(), Passed: true, Severity: SeverityInfo, Message: "Cannot read /proc/swaps"}
	}
	defer f.Close()

	// 使用 Scanner 逐行读取，避免读取整个文件（虽然通常很小）
	scanner := bufio.NewScanner(f)
	swapCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过标题行和空行
		if strings.HasPrefix(line, "Filename") || strings.TrimSpace(line) == "" {
			continue
		}
		swapCount++
	}

	if err := scanner.Err(); err != nil {
		return Result{Name: c.Name(), Passed: false, Severity: SeverityWarn, Error: err, Message: "Error reading /proc/swaps"}
	}

	if swapCount > 0 {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: c.Severity,
			Message:  "System swap is enabled. This may cause GC latency spikes.",
		}
	}

	return Result{Name: c.Name(), Passed: true}
}
