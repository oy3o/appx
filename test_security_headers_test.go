package appx

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpService_SecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello headers"))
	})

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	svc := NewHttpService("headers-svc", addr, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = svc.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	client := &http.Client{}
	resp, err := client.Get("http://" + addr)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Empty(t, resp.Header.Get("Strict-Transport-Security")) // Since TLS is not enabled here

	_ = svc.Stop(context.Background())
}
