package appx

import (
	"io"
	"net/http/httptest"
	"testing"
)

func BenchmarkWriteBytes(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.Write([]byte("OK"))
		w.Body.Reset()
	}
}

func BenchmarkWriteString(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		io.WriteString(w, "OK")
		w.Body.Reset()
	}
}
