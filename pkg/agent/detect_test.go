package agent_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/agent"
)

func TestDetect_FromBinaryName(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		args         []string
		wantType     agent.Type
		wantImgMatch string // substring the image must contain
	}{
		{[]string{"python", "-c", "print('x')"}, agent.TypePython, "python"},
		{[]string{"python3", "script.py"}, agent.TypePython, "python"},
		{[]string{"pip", "install", "numpy"}, agent.TypePython, "python"},
		{[]string{"node", "app.js"}, agent.TypeNode, "node"},
		{[]string{"npm", "test"}, agent.TypeNode, "node"},
		{[]string{"npx", "jest"}, agent.TypeNode, "node"},
		{[]string{"bun", "run", "index.ts"}, agent.TypeBun, "bun"},
		{[]string{"go", "run", "."}, agent.TypeGo, "golang"},
		{[]string{"claude", "--help"}, agent.TypeClaudeCode, "sandbox-claude"},
		{[]string{"gemini", "--help"}, agent.TypeGeminiCLI, "sandbox-gemini"},
		{[]string{"kilocode", "--help"}, agent.TypeKilocode, "sandbox-kilocode"},
		{[]string{"kilo", "--help"}, agent.TypeKilocode, "sandbox-kilocode"},
		{[]string{"codex", "--help"}, agent.TypeCodex, "sandbox-codex"},
		{[]string{"unknown-tool"}, agent.TypeGeneric, "alpine"},
	}

	for _, tc := range tests {
		t.Run(tc.args[0], func(t *testing.T) {
			a, err := agent.Detect(tc.args, map[string]string{}, map[string]string{}, logger)
			require.NoError(t, err)
			require.Equal(t, tc.wantType, a.Name)
			require.Contains(t, a.Image, tc.wantImgMatch)
		})
	}
}

func TestDetect_FromEnvVar(t *testing.T) {
	logger := zap.NewNop()

	env := map[string]string{"AGENT_NAME": "python"}
	a, err := agent.Detect([]string{"some-other-binary"}, env, map[string]string{}, logger)
	require.NoError(t, err)
	require.Equal(t, agent.TypePython, a.Name)
}

func TestDetect_EnvVarPriority(t *testing.T) {
	logger := zap.NewNop()

	// AGENT_NAME=node should win over the "python3" binary arg.
	env := map[string]string{"AGENT_NAME": "node"}
	a, err := agent.Detect([]string{"python3", "script.py"}, env, map[string]string{}, logger)
	require.NoError(t, err)
	require.Equal(t, agent.TypeNode, a.Name)
}

func TestDetect_ImageOverride(t *testing.T) {
	logger := zap.NewNop()

	overrides := map[string]string{"python": "my-custom-python:3.11"}
	a, err := agent.Detect([]string{"python"}, map[string]string{}, overrides, logger)
	require.NoError(t, err)
	require.Equal(t, "my-custom-python:3.11", a.Image)
}

func TestDetect_NoArgs(t *testing.T) {
	logger := zap.NewNop()
	// With no args, should fall back to generic.
	a, err := agent.Detect([]string{}, map[string]string{}, map[string]string{}, logger)
	require.NoError(t, err)
	require.Equal(t, agent.TypeGeneric, a.Name)
}

func TestEnvToMap(t *testing.T) {
	environ := []string{"FOO=bar", "BAZ=qux=extra", "NO_VALUE"}
	m := agent.EnvToMap(environ)
	require.Equal(t, "bar", m["FOO"])
	require.Equal(t, "qux=extra", m["BAZ"])
	require.NotContains(t, m, "NO_VALUE")
}
