package container_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	cnt "github.com/servusdei2018/sandbox/pkg/container"
)

func TestFilterEnv_WhitelistBasic(t *testing.T) {
	logger := zap.NewNop()

	hostEnv := map[string]string{
		"PATH": "/usr/bin:/bin",
		"HOME": "/root",
		"USER": "testuser",
	}

	result := cnt.FilterEnv(hostEnv, []string{"PATH", "HOME"}, []string{}, logger)
	asMap := sliceToMap(result)

	require.Equal(t, "/usr/bin:/bin", asMap["PATH"])
	require.Equal(t, "/root", asMap["HOME"])
	require.NotContains(t, asMap, "USER")
}

func TestFilterEnv_BlocklistWins(t *testing.T) {
	logger := zap.NewNop()

	hostEnv := map[string]string{
		"PATH":              "/usr/bin",
		"AWS_ACCESS_KEY":    "secret-key",
		"GITHUB_TOKEN":      "ghp_abc123",
		"ANTHROPIC_API_KEY": "sk-ant-xyz",
	}

	whitelist := []string{"PATH", "AWS_ACCESS_KEY", "GITHUB_TOKEN", "ANTHROPIC_API_KEY"}
	blocklist := []string{"AWS_*", "GITHUB_TOKEN", "ANTHROPIC_API_KEY"}

	result := cnt.FilterEnv(hostEnv, whitelist, blocklist, logger)
	asMap := sliceToMap(result)

	require.Equal(t, "/usr/bin", asMap["PATH"])
	require.NotContains(t, asMap, "AWS_ACCESS_KEY", "AWS vars must be blocked")
	require.NotContains(t, asMap, "GITHUB_TOKEN", "GITHUB_TOKEN must be blocked")
	require.NotContains(t, asMap, "ANTHROPIC_API_KEY", "ANTHROPIC_API_KEY must be blocked")
}

func TestFilterEnv_WildcardWhitelist(t *testing.T) {
	logger := zap.NewNop()

	hostEnv := map[string]string{
		"LC_ALL":  "en_US.UTF-8",
		"LC_TIME": "C",
		"LANG":    "en_US.UTF-8",
		"SECRET":  "dont-pass",
	}

	result := cnt.FilterEnv(hostEnv, []string{"LC_*", "LANG"}, []string{}, logger)
	asMap := sliceToMap(result)

	require.Equal(t, "en_US.UTF-8", asMap["LC_ALL"])
	require.Equal(t, "C", asMap["LC_TIME"])
	require.Equal(t, "en_US.UTF-8", asMap["LANG"])
	require.NotContains(t, asMap, "SECRET")
}

func TestFilterEnv_EmptyHost(t *testing.T) {
	logger := zap.NewNop()
	result := cnt.FilterEnv(map[string]string{}, []string{"PATH"}, []string{}, logger)
	require.Empty(t, result)
}

func TestFilterEnv_EmptyWhitelist(t *testing.T) {
	logger := zap.NewNop()
	hostEnv := map[string]string{"PATH": "/usr/bin", "HOME": "/root"}
	result := cnt.FilterEnv(hostEnv, []string{}, []string{}, logger)
	require.Empty(t, result)
}

// sliceToMap converts a []string of "KEY=VALUE" entries to a map.
func sliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		for i, c := range kv {
			if c == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}
