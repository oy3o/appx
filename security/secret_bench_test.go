package security

import "testing"

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
