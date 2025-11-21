//go:build !linux

package security

import "context"

// 在非 Linux 系统下，这些检查直接通过（或不做任何事）

type UlimitChecker struct {
	MinLimit uint64
	Severity Severity
}

func (c *UlimitChecker) Name() string { return "os_ulimit" }
func (c *UlimitChecker) Check(ctx context.Context) Result {
	return Result{Name: c.Name(), Passed: true, Message: "Skipped on non-linux OS"}
}

type SysctlChecker struct {
	Key      string
	MinValue int
	Severity Severity
}

func (c *SysctlChecker) Name() string { return "os_sysctl:" + c.Key }
func (c *SysctlChecker) Check(ctx context.Context) Result {
	return Result{Name: c.Name(), Passed: true, Message: "Skipped on non-linux OS"}
}
