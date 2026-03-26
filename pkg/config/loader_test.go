package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Point HOME to a temp dir so no real ~/.sandbox/config.yaml is read.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	logger := zap.NewNop()
	cfg, err := config.Load(logger)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "alpine:latest", cfg.Images["default"])
	require.Equal(t, "/work", cfg.Paths.Workspace)
	require.Equal(t, "30m", cfg.Container.Timeout)
	require.True(t, cfg.Container.Remove)
	require.Equal(t, "info", cfg.Logging.Level)
	require.Equal(t, "console", cfg.Logging.Format)
}

func TestLoad_FromFile(t *testing.T) {
	tmp := t.TempDir()
	sandboxDir := filepath.Join(tmp, ".sandbox")
	require.NoError(t, os.MkdirAll(sandboxDir, 0o755))

	cfgContent := `
logging:
  level: "debug"
  format: "json"
container:
  timeout: "5m"
  remove: false
`
	require.NoError(t, os.WriteFile(filepath.Join(sandboxDir, "config.yaml"), []byte(cfgContent), 0o644))

	t.Setenv("HOME", tmp)

	logger := zap.NewNop()
	cfg, err := config.Load(logger)
	require.NoError(t, err)
	require.Equal(t, "debug", cfg.Logging.Level)
	require.Equal(t, "json", cfg.Logging.Format)
	require.Equal(t, "5m", cfg.Container.Timeout)
	require.False(t, cfg.Container.Remove)
	// Defaults should still apply for unset keys.
	require.Equal(t, "/work", cfg.Paths.Workspace)
}

func TestLoad_MalformedFile(t *testing.T) {
	tmp := t.TempDir()
	sandboxDir := filepath.Join(tmp, ".sandbox")
	require.NoError(t, os.MkdirAll(sandboxDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sandboxDir, "config.yaml"), []byte("!!invalid: yaml: ["), 0o644))

	t.Setenv("HOME", tmp)

	logger := zap.NewNop()
	_, err := config.Load(logger)
	require.Error(t, err)
}

func TestWriteDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	logger := zap.NewNop()
	require.NoError(t, config.WriteDefault(logger))

	cfgFile := filepath.Join(tmp, ".sandbox", "config.yaml")
	_, err := os.Stat(cfgFile)
	require.NoError(t, err, "default config file should exist")

	// Calling again should be idempotent.
	require.NoError(t, config.WriteDefault(logger))
}
