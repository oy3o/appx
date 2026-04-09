package security

import (
	"strings"
	"testing"
)

func CheckWeakOld(secret string) bool {
	for _, weak := range WeakList {
		if strings.EqualFold(secret, weak) {
			return true
		}
	}
	return false
}

func CheckWeakNew(secret string) bool {
	for _, weak := range WeakList {
		if len(secret) == len(weak) && secret == weak {
			return true
		}
		if strings.EqualFold(secret, weak) {
			return true
		}
	}
	return false
}

func BenchmarkCheckWeakOld(b *testing.B) {
	secret := "qwerty"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckWeakOld(secret)
	}
}

func BenchmarkCheckWeakNew(b *testing.B) {
	secret := "qwerty"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckWeakNew(secret)
	}
}
