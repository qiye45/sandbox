package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/servusdei2018/sandbox/pkg/config"
)

func TestNormalizeMountMode(t *testing.T) {
	require.Equal(t, config.MountModeWrite, config.NormalizeMountMode(""))
	require.Equal(t, config.MountModeWrite, config.NormalizeMountMode("w"))
	require.Equal(t, config.MountModeRead, config.NormalizeMountMode("r"))
	require.Equal(t, config.MountModeRead, config.NormalizeMountMode(" R "))
	require.Equal(t, config.MountModeWrite, config.NormalizeMountMode("ro"))
	require.Equal(t, config.MountModeWrite, config.NormalizeMountMode("invalid"))
}
