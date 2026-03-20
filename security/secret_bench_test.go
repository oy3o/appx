package security

import (
	"context"
	"testing"
)

func BenchmarkCalculateEntropy(b *testing.B) {
	str := "this is a test string to calculate entropy for password strength checking"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		calculateEntropy(str)
	}
}

func BenchmarkCalculateEntropyUnicode(b *testing.B) {
	str := "this is a test string with some unicode 密码 strength checking 🔥"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		calculateEntropy(str)
	}
}

func BenchmarkCheckCommonDefault(b *testing.B) {
	checker := &SecretStrengthChecker{
		NameID:    "test",
		Secret:    "something_long_and_not_in_weak_list_at_all_but_checked_against_it",
		MinLength: 8,
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}
