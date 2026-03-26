package security

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// 这些是开发环境中极其常见的弱密码
var WeakList = []string{
	"123456", "12345678", "123456789", "password", "secret", "admin", "root",
	"changeme", "default", "actuator", "manager", "111111", "qwerty",
	"1234567890", "system", "service", "auth", "token", "key",
}

// checkComplexity checks if a string contains at least one letter and at least one number/symbol.
// Performance optimization: Combine two separate loop iterations into a single pass that can break early.
func checkComplexity(s string) (bool, bool) {
	hasLetter := false
	hasNumberOrSymbol := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !hasLetter && ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			hasLetter = true
		} else if !hasNumberOrSymbol && (c >= '0' && c <= '9' || c >= '!' && c <= '/' || c >= ':' && c <= '@' || c >= '[' && c <= '_' || c >= '{' && c <= '}') {
			hasNumberOrSymbol = true
		}
		if hasLetter && hasNumberOrSymbol {
			break
		}
	}
	return hasLetter, hasNumberOrSymbol
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
	// Performance optimization: Using direct string iteration avoids significant CPU overhead on every check compared to regexp execution.
	// We combine both checks into a single loop to avoid iterating over the string twice.
	hasLetter, hasNumberOrSymbol := checkComplexity(c.Secret)

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
		return 0.0
	}

	// 性能优化: 大多数密码仅包含 ASCII 字符，使用数组避免堆分配
	// Optimization: ASCII characters are 0-127, use a smaller array to reduce loop iterations
	var asciiCounts [128]int
	var unicodeCounts map[rune]int

	// Optimization: By manually iterating over bytes and using utf8.DecodeRuneInString for multi-byte characters,
	// we avoid the implicit utf8 decoding overhead of range for pure ASCII characters in hot paths.
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			asciiCounts[c]++
			i++
		} else {
			r, size := utf8.DecodeRuneInString(s[i:])
			if unicodeCounts == nil {
				unicodeCounts = make(map[rune]int)
			}
			unicodeCounts[r]++
			i += size
		}
	}

	length := float64(len(s))
	// Optimize: log2(length) - (1 / length) * sum(count_i * log2(count_i))
	// This avoids division inside the loop and calculates entropy faster.
	sumCountLogCount := 0.0

	for i := 0; i < len(asciiCounts); i++ {
		count := asciiCounts[i]
		if count > 0 {
			c := float64(count)
			sumCountLogCount += c * math.Log2(c)
		}
	}

	for _, count := range unicodeCounts {
		c := float64(count)
		sumCountLogCount += c * math.Log2(c)
	}

	return math.Log2(length) - (sumCountLogCount / length)
}
