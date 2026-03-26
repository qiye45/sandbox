package agent_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/servusdei2018/sandbox/pkg/agent"
)

func TestDefaultImage_KnownAgents(t *testing.T) {
	tests := []struct {
		agentType  agent.Type
		wantSubstr string
	}{
		{agent.TypePython, "python"},
		{agent.TypeNode, "node"},
		{agent.TypeBun, "bun"},
		{agent.TypeGo, "golang"},
		{agent.TypeClaudeCode, "sandbox-claude"},
		{agent.TypeGeminiCLI, "sandbox-gemini"},
		{agent.TypeKilocode, "sandbox-kilocode"},
		{agent.TypeGeneric, "alpine"},
	}
	for _, tc := range tests {
		t.Run(string(tc.agentType), func(t *testing.T) {
			img := agent.DefaultImage(tc.agentType)
			require.NotEmpty(t, img)
			require.Contains(t, img, tc.wantSubstr)
		})
	}
}

func TestDefaultImage_UnknownType(t *testing.T) {
	img := agent.DefaultImage("totally-made-up")
	require.Equal(t, "alpine:latest", img)
}

func TestOverrideImages(t *testing.T) {
	// Override and verify.
	agent.OverrideImages(map[string]string{
		"python": "custom-python:3.99",
	})
	require.Equal(t, "custom-python:3.99", agent.DefaultImage(agent.TypePython))

	// Restore default so other tests aren't affected.
	agent.OverrideImages(map[string]string{
		"python": "python:3.13-alpine",
	})
}
