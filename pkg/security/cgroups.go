package security

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// ParseMemoryBytes converts a human-readable memory string to bytes.
// Supported suffixes: B, KB, MB, GB, TB (case-insensitive).
// Returns 0 (no limit) when s is empty or "0".
func ParseMemoryBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	// Ordered longest-first so "MB" is matched before "B", etc.
	type entry struct {
		suffix string
		mult   int64
	}
	suffixes := []entry{
		{"TIB", 1 << 40},
		{"GIB", 1 << 30},
		{"MIB", 1 << 20},
		{"KIB", 1 << 10},
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	upper := strings.ToUpper(s)
	for _, e := range suffixes {
		if strings.HasSuffix(upper, e.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(upper, e.suffix))
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("cgroups: cannot parse memory value %q: %w", s, err)
			}
			return int64(n * float64(e.mult)), nil
		}
	}

	// Bare integer — treat as bytes.
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cgroups: cannot parse memory value %q: %w", s, err)
	}
	return n, nil
}

// ResourceLimitsConfig carries the parsed cgroup parameters we care about.
type ResourceLimitsConfig struct {
	// MemoryBytes is the hard memory limit; 0 means no limit.
	MemoryBytes int64
	// CPUQuota is the CFS quota in microseconds per 100ms period; 0 means unlimited.
	CPUQuota int64
	// PidsLimit is the maximum number of PIDs in the container; 0 means unlimited.
	PidsLimit int64
}

// BuildResources converts a ResourceLimitsConfig into the Docker container.Resources
// struct that is embedded in container.HostConfig.
func BuildResources(lim ResourceLimitsConfig) container.Resources {
	pids := lim.PidsLimit
	if pids < 0 {
		pids = 0
	}

	return container.Resources{
		Memory:    lim.MemoryBytes,
		CPUQuota:  lim.CPUQuota,
		PidsLimit: &pids,
	}
}
