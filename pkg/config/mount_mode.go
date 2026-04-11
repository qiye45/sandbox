package config

import "strings"

const (
	// MountModeWrite is a read-write bind mount.
	MountModeWrite = "w"
	// MountModeRead is a read-only bind mount.
	MountModeRead = "r"
)

// NormalizeMountMode returns a valid mount mode.
// Empty or unknown values default to write mode.
func NormalizeMountMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case MountModeRead:
		return MountModeRead
	case MountModeWrite, "":
		return MountModeWrite
	default:
		return MountModeWrite
	}
}
