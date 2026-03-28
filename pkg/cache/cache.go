package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// GetStats calculates the total size in bytes and number of files in a directory.
func GetStats(path string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the path doesn't exist, we treat it as 0 bytes/files instead of failing.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if !d.IsDir() {
			fileCount++
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate cache stats: %w", err)
	}

	return totalSize, fileCount, nil
}

// Clean removes all contents of the directory at path but keeps the directory itself.
func Clean(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
	}

	return nil
}

// FormatSize converts bytes to a human-readable string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
