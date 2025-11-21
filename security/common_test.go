package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePermChecker(t *testing.T) {
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.key")

	// 创建一个文件
	err := os.WriteFile(secretFile, []byte("content"), 0o600)
	require.NoError(t, err)

	t.Run("Should pass on correct permissions", func(t *testing.T) {
		c := &FilePermChecker{
			Path:    secretFile,
			MaxPerm: 0o600,
		}
		res := c.Check(context.Background())
		assert.True(t, res.Passed)
	})

	t.Run("Should fail on loose permissions", func(t *testing.T) {
		// 修改为 0644 (Group/Other 可读)
		err := os.Chmod(secretFile, 0o644)
		require.NoError(t, err)

		c := &FilePermChecker{
			Path:    secretFile,
			MaxPerm: 0o600,
		}
		res := c.Check(context.Background())
		assert.False(t, res.Passed)
		assert.Contains(t, res.Message, "Insecure permissions")
	})

	t.Run("Should fail if file does not exist", func(t *testing.T) {
		c := &FilePermChecker{
			Path:    filepath.Join(tmpDir, "missing.file"),
			MaxPerm: 0o600,
		}
		res := c.Check(context.Background())
		assert.False(t, res.Passed)
		assert.Contains(t, res.Message, "not found")
	})
}

func TestConfigChecker(t *testing.T) {
	t.Run("Should execute check function", func(t *testing.T) {
		c := &ConfigChecker{
			ID: "test",
			CheckFn: func() (bool, string) {
				return false, "config is wrong"
			},
		}
		res := c.Check(context.Background())
		assert.False(t, res.Passed)
		assert.Equal(t, "config is wrong", res.Message)
	})
}

func TestRootUserChecker(t *testing.T) {
	// 在普通单元测试环境中，我们通常不是 Root。
	// 所以这里测试 "Non-Root" 通过的情况。
	if os.Geteuid() == 0 {
		t.Skip("Skipping RootUserChecker test because tests are running as root")
	}

	c := &RootUserChecker{}
	res := c.Check(context.Background())
	assert.True(t, res.Passed)
}
