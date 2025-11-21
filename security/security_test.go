package security

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// MockChecker 用于测试 Manager 行为的桩
type MockChecker struct {
	NameVal   string
	ResultVal Result
}

func (m *MockChecker) Name() string                   { return m.NameVal }
func (m *MockChecker) Check(_ context.Context) Result { return m.ResultVal }

func TestManager_Run(t *testing.T) {
	logger := &log.Logger

	t.Run("Should pass when all checkers pass", func(t *testing.T) {
		mgr := New(logger)
		mgr.Register(&MockChecker{
			NameVal:   "success_check",
			ResultVal: Result{Name: "success_check", Passed: true},
		})

		err := mgr.Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Should NOT return error on Info/Warn failures", func(t *testing.T) {
		mgr := New(logger)
		mgr.Register(
			&MockChecker{
				NameVal:   "warn_check",
				ResultVal: Result{Name: "warn_check", Passed: false, Severity: SeverityWarn, Message: "warning"},
			},
			&MockChecker{
				NameVal:   "info_check",
				ResultVal: Result{Name: "info_check", Passed: false, Severity: SeverityInfo, Message: "info"},
			},
		)

		err := mgr.Run(context.Background())
		assert.NoError(t, err, "Manager should only return error on Fatal severity")
	})

	t.Run("Should return error on Fatal failure", func(t *testing.T) {
		mgr := New(logger)
		mgr.Register(&MockChecker{
			NameVal:   "fatal_check",
			ResultVal: Result{Name: "fatal_check", Passed: false, Severity: SeverityFatal, Message: "boom", Error: errors.New("oops")},
		})

		err := mgr.Run(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fatal errors found")
	})
}
