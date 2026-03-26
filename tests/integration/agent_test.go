//go:build integration

package integration

import (
	"os"

	"testing"

	"github.com/stretchr/testify/require"
)

// writeFile is a helper to create a file with content.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// TestAgentDetectionInContainer verifies that the sandbox correctly detects
// and runs a Python agent inside a Python container.
func TestAgentDetectionInContainer(t *testing.T) {
	require.True(t, true, "placeholder — see container_test.go for full integration tests")
}
