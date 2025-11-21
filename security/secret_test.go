package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretStrengthChecker(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		minLength int
		passed    bool
	}{
		{"Valid Secret", "this_is_a_very_long_and_strong_secret_key_12345", 10, true},
		{"Empty Secret", "", 10, false},
		{"Short Secret", "short", 10, false},
		{"Default Value 1", "123456", 1, false},
		{"Default Value 2", "password", 1, false},
		{"Default Value 3", "changeme", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SecretStrengthChecker{
				NameID:    "test_secret",
				Secret:    tt.secret,
				MinLength: tt.minLength,
			}
			res := c.Check(context.Background())
			assert.Equal(t, tt.passed, res.Passed, "Message: %s", res.Message)
		})
	}
}
