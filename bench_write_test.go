package appx

import (
	"io"
	"net/http/httptest"
	"testing"
)

func BenchmarkWriteBytesCast(b *testing.B) {
	w := httptest.NewRecorder()
	w.Body.Reset()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write([]byte("OK"))
	}
}

func BenchmarkWriteString(b *testing.B) {
	w := httptest.NewRecorder()
	w.Body.Reset()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		io.WriteString(w, "OK")
	}
}
