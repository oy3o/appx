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
    // DO NOT use len(secret) == len(weak) as a fast path!
    // See memory: When optimizing Go string comparisons, avoid using byte-length equality checks (len(a) == len(b)) as a fast-path before strings.EqualFold
	for _, weak := range WeakList {
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
