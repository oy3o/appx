package security

import (
	"context"
	"fmt"
	"strings"
)

// BindAddrChecker 检查监听地址是否过于宽泛
type BindAddrChecker struct {
	Addr        string
	AllowPublic bool // 是否允许公网暴露
}

func (c *BindAddrChecker) Name() string { return "network_bind:" + c.Addr }

func (c *BindAddrChecker) Check(ctx context.Context) Result {
	// 检查 IPv4 全零, IPv6 全零, 以及省略 IP 的简写模式
	isPublic := strings.Contains(c.Addr, "0.0.0.0") ||
		strings.Contains(c.Addr, "[::]") ||
		strings.HasPrefix(c.Addr, ":")

	if isPublic {
		if !c.AllowPublic {
			return Result{
				Name:     c.Name(),
				Passed:   false,
				Severity: SeverityWarn,
				Message:  fmt.Sprintf("Service is listening on all interfaces (%s). Ensure this is intended.", c.Addr),
			}
		}
	}
	return Result{Name: c.Name(), Passed: true}
}
