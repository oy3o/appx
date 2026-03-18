package security

import (
	"context"
	"testing"
)

func BenchmarkSecretStrengthChecker(b *testing.B) {
	checker := &SecretStrengthChecker{
		NameID: "test",
		Secret: "StrongP@ssw0rd!",
	}
	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}
