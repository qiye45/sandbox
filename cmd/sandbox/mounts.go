package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

const hostPathMountRoot = "/sandbox-host-paths"

func workspaceGitMaskBind(workspaceDir string, mountTarget string, tmpRoot string) (string, func(), error) {
	noop := func() {}

	gitDir := filepath.Join(workspaceDir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", noop, nil
		}
		return "", noop, fmt.Errorf("failed to inspect workspace .git directory: %w", err)
	}
	if !info.IsDir() {
		return "", noop, nil
	}

	if tmpRoot == "" {
		tmpRoot = os.TempDir()
	}
	if err := os.MkdirAll(tmpRoot, 0o700); err != nil {
		return "", noop, fmt.Errorf("failed to create git mask temp root: %w", err)
	}

	maskDir, err := os.MkdirTemp(tmpRoot, "git-mask-*")
	if err != nil {
		return "", noop, fmt.Errorf("failed to create git mask directory: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(maskDir)
	}

	target := path.Join(normalizeContainerPath(mountTarget), ".git")
	return fmt.Sprintf("%s:%s:ro", maskDir, target), cleanup, nil
}

func normalizeContainerPath(target string) string {
	cleaned := strings.TrimSpace(target)
	if cleaned == "" {
		return "/work"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return path.Clean(cleaned)
}

// hostPathReadonlyBinds converts host PATH entries into read-only bind mounts.
// Each valid absolute directory is mounted under an isolated container prefix
// to avoid shadowing paths that already exist inside the image.
func hostPathReadonlyBinds(pathEnv string, logger *zap.Logger) ([]string, []string) {
	entries := filepath.SplitList(pathEnv)
	binds := make([]string, 0, len(entries))
	containerPathEntries := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if !filepath.IsAbs(entry) {
			if logger != nil {
				logger.Debug("skipping non-absolute PATH entry", zap.String("entry", entry))
			}
			continue
		}

		resolved := filepath.Clean(entry)
		if symlinkResolved, err := filepath.EvalSymlinks(resolved); err == nil {
			resolved = symlinkResolved
		}

		info, err := os.Stat(resolved)
		if err != nil {
			if logger != nil {
				logger.Debug("skipping PATH entry because it does not exist", zap.String("entry", resolved), zap.Error(err))
			}
			continue
		}
		if !info.IsDir() {
			if logger != nil {
				logger.Debug("skipping PATH entry because it is not a directory", zap.String("entry", resolved))
			}
			continue
		}

		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}

		containerPath := path.Join(hostPathMountRoot, fmt.Sprintf("%d", len(containerPathEntries)))
		binds = append(binds, fmt.Sprintf("%s:%s:ro", resolved, containerPath))
		containerPathEntries = append(containerPathEntries, containerPath)
	}

	return binds, containerPathEntries
}
