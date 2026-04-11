package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorkspaceGitMaskBind_NoGitDir(t *testing.T) {
	workspace := t.TempDir()
	tmpRoot := filepath.Join(t.TempDir(), "tmp")

	bind, cleanup, err := workspaceGitMaskBind(workspace, "/work", tmpRoot)
	require.NoError(t, err)
	require.Empty(t, bind)
	require.NotNil(t, cleanup)
	cleanup()
}

func TestWorkspaceGitMaskBind_WithGitDir(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(workspace, ".git"), 0o755))
	tmpRoot := filepath.Join(t.TempDir(), "tmp")

	bind, cleanup, err := workspaceGitMaskBind(workspace, "/work", tmpRoot)
	require.NoError(t, err)
	require.NotEmpty(t, bind)

	source, target, mode := splitBindSpec(t, bind)
	require.Equal(t, "/work/.git", target)
	require.Equal(t, "ro", mode)

	info, err := os.Stat(source)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	cleanup()
	_, err = os.Stat(source)
	require.True(t, os.IsNotExist(err))
}

func TestWorkspaceGitMaskBind_CustomMountTarget(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(workspace, ".git"), 0o755))

	bind, cleanup, err := workspaceGitMaskBind(workspace, "repo", "")
	require.NoError(t, err)
	defer cleanup()

	_, target, mode := splitBindSpec(t, bind)
	require.Equal(t, "/repo/.git", target)
	require.Equal(t, "ro", mode)
}

func TestNormalizeContainerPath(t *testing.T) {
	require.Equal(t, "/work", normalizeContainerPath(""))
	require.Equal(t, "/work", normalizeContainerPath("/work"))
	require.Equal(t, "/repo", normalizeContainerPath("repo"))
	require.Equal(t, "/repo", normalizeContainerPath("/repo/"))
}

func TestHostPathReadonlyBinds(t *testing.T) {
	tmp := t.TempDir()
	binA := filepath.Join(tmp, "bin-a")
	binB := filepath.Join(tmp, "bin-b")
	notDir := filepath.Join(tmp, "not-dir")
	missing := filepath.Join(tmp, "missing")
	linkToA := filepath.Join(tmp, "link-a")

	require.NoError(t, os.MkdirAll(binA, 0o755))
	require.NoError(t, os.MkdirAll(binB, 0o755))
	require.NoError(t, os.WriteFile(notDir, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(binA, linkToA))

	pathEnv := strings.Join([]string{
		binA,
		"",
		"relative/bin",
		missing,
		notDir,
		linkToA,
		binB,
	}, string(os.PathListSeparator))

	containerBinds, containerPathEntries := hostPathReadonlyBinds(pathEnv, zap.NewNop())
	require.Equal(t, []string{
		fmt.Sprintf("%s:/sandbox-host-paths/0:ro", binA),
		fmt.Sprintf("%s:/sandbox-host-paths/1:ro", binB),
	}, containerBinds)
	require.Equal(t, []string{"/sandbox-host-paths/0", "/sandbox-host-paths/1"}, containerPathEntries)
}

func splitBindSpec(t *testing.T, bind string) (string, string, string) {
	t.Helper()
	parts := strings.Split(bind, ":")
	require.GreaterOrEqual(t, len(parts), 3)

	mode := parts[len(parts)-1]
	target := parts[len(parts)-2]
	source := strings.Join(parts[:len(parts)-2], ":")
	return source, target, mode
}
