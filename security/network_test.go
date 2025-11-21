package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBindAddrChecker(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		allowPublic bool
		passed      bool
	}{
		{"Localhost Allowed", "127.0.0.1:8080", false, true},
		{"Internal IP Allowed", "192.168.1.5:8080", false, true},

		{"Public Bind Blocked", "0.0.0.0:8080", false, false},
		{"Public Bind Allowed", "0.0.0.0:8080", true, true},

		{"Short Port Blocked", ":8080", false, false},
		{"Short Port Allowed", ":8080", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &BindAddrChecker{
				Addr:        tt.addr,
				AllowPublic: tt.allowPublic,
			}
			res := c.Check(context.Background())
			assert.Equal(t, tt.passed, res.Passed, "Message: %s", res.Message)
		})
	}
}
