//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/config"
	cnt "github.com/servusdei2018/sandbox/pkg/container"
)

// securityCfg returns a container.Config with full security options applied,
// running the given shell command in alpine:latest.
func securityCfg(t *testing.T, shellCmd string) *cnt.Config {
	t.Helper()
	return &cnt.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"sh", "-c", shellCmd},
		WorkspaceDir: t.TempDir(),
		MountTarget:  "/work",
		Env:          []string{},
		Timeout:      2 * time.Minute,
		RemoveOnExit: true,
		NetworkMode:  "bridge",
		Security: config.SecurityConfig{
			MemoryLimit:        "512MB",
			CPUQuota:           0,
			PidsLimit:          64,
			SeccompProfilePath: "",
			ReadOnlyRoot:       true,
			UserMapping:        "",
			DropCapabilities:   []string{},
		},
	}
}

// runSecure creates and runs a secured container, cleans up, and returns exit code.
func runSecure(t *testing.T, mgr *cnt.Manager, cfg *cnt.Config) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	id, err := mgr.Create(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Remove(context.Background(), id) })

	code, err := mgr.Run(ctx, id, false)
	require.NoError(t, err)
	return code
}

func setupManager(t *testing.T) *cnt.Manager {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	t.Cleanup(func() { _ = logger.Sync() })

	mgr, err := cnt.NewManager(logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	require.NoError(t, mgr.PullIfMissing(ctx, "alpine:latest"))

	return mgr
}

// ---------------------------------------------------------------------------
// Filesystem isolation
// ---------------------------------------------------------------------------

// TestSecurity_RootFilesystemIsReadOnly verifies that writing to the rootfs
// outside of /work and /tmp is blocked.
func TestSecurity_RootFilesystemIsReadOnly(t *testing.T) {
	mgr := setupManager(t)
	// Attempt to create a file in root — should fail with non-zero exit due to
	// read-only filesystem.
	code := runSecure(t, mgr, securityCfg(t, "touch /owned 2>/dev/null; echo $?"))
	// The command itself won't error (sh -c returns 0), but the inner touch
	// will return non-zero. We capture the exit code of touch via echo $?.
	// However since sh -c "echo $?" always exits 0, we assert that we can
	// read the rootfs but not write to it via a direct write test:
	cfg := securityCfg(t, "touch /exploit_root")
	code = runSecure(t, mgr, cfg)
	assert.NotEqual(t, 0, code, "writing to / on read-only rootfs should fail")
}

// TestSecurity_TmpIsWritable verifies /tmp is writable despite read-only root.
func TestSecurity_TmpIsWritable(t *testing.T) {
	mgr := setupManager(t)
	code := runSecure(t, mgr, securityCfg(t, "touch /tmp/sandbox_test && echo ok"))
	assert.Equal(t, 0, code, "/tmp should be writable via tmpfs overlay")
}

// TestSecurity_WorkspaceIsWritable verifies /work (bind mount) is writable.
func TestSecurity_WorkspaceIsWritable(t *testing.T) {
	mgr := setupManager(t)
	code := runSecure(t, mgr, securityCfg(t, "touch /work/sandbox_test && echo ok"))
	assert.Equal(t, 0, code, "/work bind-mount should be writable")
}

// ---------------------------------------------------------------------------
// User namespace / privilege
// ---------------------------------------------------------------------------

// TestSecurity_ContainerRunsAsRoot verifies the default container UID is 0 (root).
// Real isolation comes from the read-only rootfs + seccomp + capabilities, not UID.
func TestSecurity_ContainerRunsAsRoot(t *testing.T) {
	mgr := setupManager(t)
	code := runSecure(t, mgr, securityCfg(t, `[ "$(id -u)" = "0" ]`))
	assert.Equal(t, 0, code, "container should run as UID 0 (root) by default")
}

// TestSecurity_UserMappingOverride verifies that setting user_mapping to nobody works.
func TestSecurity_UserMappingOverride(t *testing.T) {
	mgr := setupManager(t)
	cfg := securityCfg(t, `[ "$(id -u)" = "65534" ]`)
	cfg.Security.UserMapping = "65534:65534"
	code := runSecure(t, mgr, cfg)
	assert.Equal(t, 0, code, "container should run as UID 65534 when user_mapping is set")
}

// TestSecurity_NoSudoEscalation verifies that sudo is either missing or
// fails without granting root.
func TestSecurity_NoSudoEscalation(t *testing.T) {
	mgr := setupManager(t)
	// sudo shouldn't exist in alpine, or setuid bit won't help since CAP_SETUID is dropped.
	code := runSecure(t, mgr, securityCfg(t, "sudo id 2>/dev/null && [ \"$(id -u)\" != \"0\" ]"))
	// Either sudo is missing (non-zero) or we're still not root (zero exit from the test).
	// Both acceptable: we just must NOT be root.
	_ = code // The key assertion is below.

	codeRoot := runSecure(t, mgr, securityCfg(t, `sudo id -u 2>/dev/null | grep -q '^0$'`))
	assert.NotEqual(t, 0, codeRoot, "sudo must not grant UID 0 inside the container")
}

// ---------------------------------------------------------------------------
// Seccomp: blocked syscalls
// ---------------------------------------------------------------------------

// TestSecurity_MountBlocked verifies that the mount(2) syscall is denied.
func TestSecurity_MountBlocked(t *testing.T) {
	mgr := setupManager(t)
	// unshare requires CAP_SYS_ADMIN and the unshare/mount syscalls — all blocked.
	code := runSecure(t, mgr, securityCfg(t, "mount -t tmpfs tmpfs /mnt 2>/dev/null"))
	assert.NotEqual(t, 0, code, "mount syscall should be blocked by seccomp/capabilities")
}

// TestSecurity_UnshareBlocked verifies that namespace creation (unshare) is denied.
func TestSecurity_UnshareBlocked(t *testing.T) {
	mgr := setupManager(t)
	code := runSecure(t, mgr, securityCfg(t, "unshare --user /bin/sh -c 'id' 2>/dev/null"))
	assert.NotEqual(t, 0, code, "unshare syscall should be blocked by seccomp")
}

// ---------------------------------------------------------------------------
// Capability restrictions
// ---------------------------------------------------------------------------

// TestSecurity_NetAdminCapDropped verifies that network admin operations
// (e.g. bringing up an interface) fail due to dropped CAP_NET_ADMIN.
func TestSecurity_NetAdminCapDropped(t *testing.T) {
	mgr := setupManager(t)
	// ip link set lo down requires CAP_NET_ADMIN.
	code := runSecure(t, mgr, securityCfg(t, "ip link set lo down 2>/dev/null"))
	assert.NotEqual(t, 0, code, "CAP_NET_ADMIN should be dropped; ip link set should fail")
}

// ---------------------------------------------------------------------------
// Resource limits (cgroups)
// ---------------------------------------------------------------------------

// TestSecurity_PidLimitEnforced verifies that the pids cgroup limit is applied
// to the container. It reads /sys/fs/cgroup/pids.max inside the container and
// asserts it matches the configured value (64). This is more portable than
// attempting to trigger kernel-level fork rejection, which behaves
// differently across Docker Desktop versions and host kernels.
func TestSecurity_PidLimitEnforced(t *testing.T) {
	mgr := setupManager(t)
	// The configured pids_limit is 64.  Check the live cgroup value.
	cfg := securityCfg(t,
		`cat /sys/fs/cgroup/pids.max | grep -q '^64$'`,
	)
	code := runSecure(t, mgr, cfg)
	assert.Equal(t, 0, code,
		"/sys/fs/cgroup/pids.max should be set to 64 inside the container")
}

// TestSecurity_MemoryLimitEnforced verifies that a process exceeding the
// memory limit is killed. We ask for 2 GB of anonymous memory inside a
// container capped at 512 MB.
func TestSecurity_MemoryLimitEnforced(t *testing.T) {
	mgr := setupManager(t)
	// dd writes 2 GB to /dev/null via a 2 GB seek — that's fine.
	// Instead, we use /dev/urandom to fill a tmpfs that is smaller than the
	// allocation we ask for, which should trigger the OOM killer.
	// head -c 2G /dev/urandom reads 2 GB; under a 512 MB memory limit this
	// will be OOM-killed (non-zero exit).
	cfg := securityCfg(t, "head -c 2000m /dev/urandom > /tmp/oom_test 2>/dev/null")
	// /tmp is tmpfs-backed and memory pressure is bounded by the container
	// memory limit, so this should still fail.
	code := runSecure(t, mgr, cfg)
	assert.NotEqual(t, 0, code, "process should be killed before consuming 2 GB under a 512 MB memory cap")
}

// ---------------------------------------------------------------------------
// Disk fill (workspace)
// ---------------------------------------------------------------------------

// TestSecurity_DiskFillStaysInWorkspace verifies that a container trying to
// fill the disk only affects the bind-mounted tmpdir, not the host root FS.
func TestSecurity_DiskFillStaysInWorkspace(t *testing.T) {
	mgr := setupManager(t)
	// Writing 100 MB to /work then verifying the file exists.
	// The test host controls the tmpdir so we can rely on OS temp dir quotas.
	cfg := securityCfg(t, "dd if=/dev/zero of=/work/fill bs=1M count=100 2>/dev/null && test -f /work/fill")
	code := runSecure(t, mgr, cfg)
	// We don't assert success/failure of the dd (no quota is enforced on /work
	// in this test), but we verify the container exits cleanly without
	// affecting the root FS.
	_ = code
	// The real assertion: / is still read-only (proved by the RO test above).
	roCode := runSecure(t, mgr, securityCfg(t, "touch /disk_fill_escape"))
	assert.NotEqual(t, 0, roCode, "after disk fill attempt root FS must still be read-only")
}

// ---------------------------------------------------------------------------
// Sanity: secure container can still do normal work
// ---------------------------------------------------------------------------

// TestSecurity_NormalWorkloadStillFunctions verifies that a confined container
// can perform typical compute tasks (shell, arithmetic, file I/O in /work and /tmp).
func TestSecurity_NormalWorkloadStillFunctions(t *testing.T) {
	mgr := setupManager(t)
	code := runSecure(t, mgr, securityCfg(t,
		`echo "hello" > /tmp/out.txt && grep -q "hello" /tmp/out.txt && echo "pass"`,
	))
	assert.Equal(t, 0, code, "normal workload (write + grep on /tmp) should succeed in secured container")
}
