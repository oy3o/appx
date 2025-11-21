//go:build linux

package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 冒烟测试：验证 UlimitChecker 在当前环境下能正常运行并返回结果
// 结果 Pass 还是 Fail 取决于运行测试的机器配置
func TestUlimitChecker_Smoke(t *testing.T) {
	c := &UlimitChecker{MinLimit: 1024}
	res := c.Check(context.Background())

	// 只要不 Panic 且返回了结果结构体，就算代码逻辑通了
	assert.Equal(t, "os_ulimit", res.Name)
	if !res.Passed {
		t.Logf("Ulimit check failed (expected on some dev machines): %s", res.Message)
	}
}

// 冒烟测试：验证 SysctlChecker
func TestSysctlChecker_Smoke(t *testing.T) {
	// 检查一个几乎所有 Linux 都有的参数
	c := &SysctlChecker{Key: "net.ipv4.ip_forward", MinValue: 0}
	res := c.Check(context.Background())

	assert.Contains(t, res.Name, "os_sysctl")
	// 这里我们不做 Passed 断言，因为容器内可能没权限读 /proc/sys
	if !res.Passed {
		t.Logf("Sysctl check failed: %s", res.Message)
	}
}

// 冒烟测试：SwapChecker
func TestSwapChecker_Smoke(t *testing.T) {
	c := &SwapChecker{}
	res := c.Check(context.Background())
	assert.Equal(t, "os_swap", res.Name)
}
