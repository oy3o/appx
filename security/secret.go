package security

import (
	"context"
	"fmt"
	"strings"
)

// SecretStrengthChecker 检查敏感字符串的强度
type SecretStrengthChecker struct {
	NameID    string
	Secret    string
	MinLength int
	// CommonWeakPasswords 可以是一个黑名单
}

func (c *SecretStrengthChecker) Name() string { return "secret_strength:" + c.NameID }

func (c *SecretStrengthChecker) Check(ctx context.Context) Result {
	if c.Secret == "" {
		return Result{
			Name: c.Name(), Passed: false, Severity: SeverityFatal,
			Message: "Secret is empty!",
		}
	}

	// 1. 检查长度
	if len(c.Secret) < c.MinLength {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: SeverityFatal,
			Message:  fmt.Sprintf("Secret is too short (%d chars), must be at least %d chars.", len(c.Secret), c.MinLength),
		}
	}

	// 2. 检查常见默认值 (黑名单)
	weakList := []string{"123456", "password", "secret", "admin", "changeme", "default"}
	for _, weak := range weakList {
		if strings.EqualFold(c.Secret, weak) {
			return Result{
				Name:     c.Name(),
				Passed:   false,
				Severity: SeverityFatal,
				Message:  fmt.Sprintf("Secret uses a common weak value: '%s'", weak),
			}
		}
	}

	return Result{Name: c.Name(), Passed: true}
}
