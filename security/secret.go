package security

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// 这些是开发环境中极其常见的弱密码
var WeakList = []string{
	"123456", "12345678", "123456789", "password", "secret", "admin", "root",
	"changeme", "default", "actuator", "manager", "111111", "qwerty",
	"1234567890", "system", "service", "auth", "token", "key",
}

// SecretStrengthChecker 检查敏感字符串的强度
type SecretStrengthChecker struct {
	NameID    string
	Secret    string
	MinLength int
	// MinEntropy 最小熵值（建议值：3.0 左右）
	MinEntropy float64
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
	minLength := c.MinLength
	if minLength == 0 {
		minLength = 8 // 默认最小长度
	}
	if len(c.Secret) < minLength {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: SeverityFatal,
			Message:  fmt.Sprintf("Secret is too short (%d chars), must be at least %d chars.", len(c.Secret), minLength),
		}
	}

	// 2. 检查常见默认值 (扩展黑名单)
	for _, weak := range WeakList {
		if strings.EqualFold(c.Secret, weak) {
			return Result{
				Name:     c.Name(),
				Passed:   false,
				Severity: SeverityFatal,
				Message:  fmt.Sprintf("Secret uses a common weak value: '%s'", weak),
			}
		}
	}

	// 3. 熵值检查 (Shannon Entropy)
	// 简单的长度检查不足以防御 "aaaaaaaa" 这种密码
	entropy := calculateEntropy(c.Secret)
	minEntropy := c.MinEntropy
	if minEntropy == 0 {
		minEntropy = 2.5 // 默认熵值阈值，"12345678" 约为 2.0，随机 8 字符约为 4.0
	}

	if entropy < minEntropy {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: SeverityFatal,
			Message:  fmt.Sprintf("Secret entropy is too low (%.2f < %.2f). Avoid repeating characters or simple sequences.", entropy, minEntropy),
		}
	}

	// 4. 复杂度检查 (包含数字和字母)
	// 简单的正则：必须包含至少一个数字或符号，且包含字母
	// 这防止了纯数字或纯字母的简单组合
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(c.Secret)
	hasNumberOrSymbol := regexp.MustCompile(`[0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(c.Secret)

	if !hasLetter || !hasNumberOrSymbol {
		return Result{
			Name:     c.Name(),
			Passed:   false,
			Severity: SeverityWarn, // 复杂度不足通常作为警告，不阻断启动（除非特别严格）
			Message:  "Secret should contain a mix of letters and numbers/symbols",
		}
	}

	return Result{Name: c.Name(), Passed: true}
}

// calculateEntropy 计算字符串的香农熵
func calculateEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	counts := make(map[rune]int)
	for _, c := range s {
		counts[c]++
	}

	entropy := 0.0
	length := float64(len(s))
	for _, count := range counts {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}
