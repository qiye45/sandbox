package container_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cnt "github.com/servusdei2018/sandbox/pkg/container"
)

func TestSelectImage_ExactMatch(t *testing.T) {
	images := map[string]string{
		"python":  "python:3.12-alpine",
		"node":    "node:22-alpine",
		"default": "alpine:latest",
	}
	require.Equal(t, "python:3.12-alpine", cnt.SelectImage("python", images))
	require.Equal(t, "node:22-alpine", cnt.SelectImage("node", images))
}

func TestSelectImage_PrefixMatch(t *testing.T) {
	images := map[string]string{
		"claude": "node:22-alpine",
	}
	require.Equal(t, "node:22-alpine", cnt.SelectImage("claude-code", images))
}

func TestSelectImage_DefaultFallback(t *testing.T) {
	images := map[string]string{
		"default": "alpine:3.19",
	}
	require.Equal(t, "alpine:3.19", cnt.SelectImage("unknown-thing", images))
}

func TestSelectImage_NilMap(t *testing.T) {
	require.Equal(t, "alpine:latest", cnt.SelectImage("anything", nil))
}

func TestSelectImage_EmptyMap(t *testing.T) {
	require.Equal(t, "alpine:latest", cnt.SelectImage("anything", map[string]string{}))
}
