package appx

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkHealthHandlerEmpty(b *testing.B) {
	app := New()
	handler := app.HealthHandler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
		w.Body.Reset()
	}
}
