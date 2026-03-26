// Package container provides Docker lifecycle management for sandbox containers.
package container

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// FilterEnv constructs the environment variable slice to be injected into the
// sandbox container. It applies the following logic in order:
//
//  1. Include any host env var whose name matches a whitelist entry (exact or
//     wildcard glob like "LC_*").
//  2. Remove any entry whose name matches a blocklist entry (blocklist wins).
//
// The returned slice has the "KEY=VALUE" format expected by the Docker API.
func FilterEnv(hostEnv map[string]string, whitelist, blocklist []string, logger *zap.Logger) []string {
	logger.Debug("filtering environment variables",
		zap.Int("host_env_count", len(hostEnv)),
		zap.Strings("whitelist", whitelist),
		zap.Strings("blocklist", blocklist),
	)

	filtered := make(map[string]string)

	for key, val := range hostEnv {
		if matchesGlob(key, whitelist) {
			filtered[key] = val
		}
	}

	for key := range filtered {
		if matchesGlob(key, blocklist) {
			delete(filtered, key)
			logger.Debug("blocked env var", zap.String("key", key))
		}
	}

	// Convert to Docker-expected "KEY=VALUE" slice.
	result := make([]string, 0, len(filtered))
	allowedKeys := make([]string, 0, len(filtered))
	for key, val := range filtered {
		result = append(result, fmt.Sprintf("%s=%s", key, val))
		allowedKeys = append(allowedKeys, key)
	}

	logger.Info("environment variables filtered",
		zap.Int("filtered_count", len(result)),
		zap.Strings("allowed_vars", allowedKeys),
	)

	return result
}

// matchesGlob returns true if name matches any pattern in patterns.
// Patterns may contain a single trailing wildcard, e.g. "AWS_*".
func matchesGlob(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		// filepath.Match is case-sensitive; also try upper-case comparison for
		// convenience (env var names are conventionally upper-case).
		if matched, _ := filepath.Match(strings.ToUpper(pattern), strings.ToUpper(name)); matched {
			return true
		}
	}
	return false
}
