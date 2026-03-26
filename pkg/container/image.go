// Package container provides Docker lifecycle management for sandbox containers.
package container

import (
	"strings"
)

// SelectImage chooses the best Docker image for the given agent name, using the
// imageMap (from config) as the first priority and falling back to DetectFromArgs.
// imageName is the 'images.<key>' config key, e.g. "python" or "claude".
func SelectImage(agentName string, imageMap map[string]string) string {
	if imageMap == nil {
		return "alpine:latest"
	}

	// Try an exact match on the agent name.
	if img, ok := imageMap[agentName]; ok && img != "" {
		return img
	}

	// Try a prefix match.
	for k, img := range imageMap {
		if strings.HasPrefix(agentName, k) && img != "" {
			return img
		}
	}

	// Fallback to the "default" image, and then to alpine.
	if dflt, ok := imageMap["default"]; ok && dflt != "" {
		return dflt
	}
	return "alpine:latest"
}
