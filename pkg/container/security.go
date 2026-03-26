package container

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"

	"github.com/servusdei2018/sandbox/pkg/config"
	"github.com/servusdei2018/sandbox/pkg/security"
)

// baseDropCaps is the minimum set of Linux capabilities we always drop,
// regardless of per-profile overrides. These grant kernel-level powers that
// are almost never needed by a compute workload and carry significant risk.
var baseDropCaps = []string{
	"CAP_NET_ADMIN",       // reconfigure network interfaces, routing
	"CAP_SYS_ADMIN",       // extremely broad: mount, bpf, many more
	"CAP_SYS_MODULE",      // load/unload kernel modules
	"CAP_SYS_PTRACE",      // ptrace, process memory inspection
	"CAP_SYS_RAWIO",       // raw I/O port access
	"CAP_SYS_BOOT",        // reboot, load new kernel
	"CAP_SYS_TIME",        // set system clock
	"CAP_AUDIT_WRITE",     // write to kernel audit log
	"CAP_AUDIT_CONTROL",   // configure audit subsystem
	"CAP_SYS_PACCT",       // process accounting
	"CAP_SYS_CHROOT",      // chroot into arbitrary dir
	"CAP_SETUID",          // set arbitrary UIDs
	"CAP_SETGID",          // set arbitrary GIDs
	"CAP_SYS_NICE",        // raise process priority beyond normal
	"CAP_NET_RAW",         // raw sockets (ping, packet sniffing)
	"CAP_MKNOD",           // create device nodes
	"CAP_LINUX_IMMUTABLE", // chattr +i on files
}

// SecurityOptions is the partial HostConfig and ContainerConfig overlay that
// ApplySecurityConfig returns. Callers merge these into the full configs.
type SecurityOptions struct {
	// HostConfig fields
	CapDrop        strslice.StrSlice
	SecurityOpt    []string
	ReadonlyRootfs bool
	Tmpfs          map[string]string
	Resources      container.Resources

	// ContainerConfig fields
	User string
}

// BuildSecurityOptions derives a SecurityOptions from the application's
// SecurityConfig. It loads (or generates) the seccomp profile, parses resource
// limits, and assembles the capability drop list.
func BuildSecurityOptions(cfg config.SecurityConfig) (SecurityOptions, error) {
	// --- Seccomp ---
	profileJSON, err := security.LoadOrDefault(cfg.SeccompProfilePath)
	if err != nil {
		return SecurityOptions{}, fmt.Errorf("container security: %w", err)
	}
	// Docker accepts the profile inline via security-opt.
	seccompOpt := fmt.Sprintf("seccomp=%s", profileJSON)

	// --- Capabilities ---
	dropSet := make(map[string]struct{}, len(baseDropCaps)+len(cfg.DropCapabilities))
	for _, cap := range baseDropCaps {
		dropSet[cap] = struct{}{}
	}
	for _, cap := range cfg.DropCapabilities {
		dropSet[cap] = struct{}{}
	}
	capDrop := make(strslice.StrSlice, 0, len(dropSet))
	for cap := range dropSet {
		capDrop = append(capDrop, cap)
	}

	// --- Resource limits ---
	memBytes, err := security.ParseMemoryBytes(cfg.MemoryLimit)
	if err != nil {
		return SecurityOptions{}, fmt.Errorf("container security: %w", err)
	}
	resources := security.BuildResources(security.ResourceLimitsConfig{
		MemoryBytes: memBytes,
		CPUQuota:    cfg.CPUQuota,
		PidsLimit:   cfg.PidsLimit,
	})

	// --- Tmpfs mounts ---
	// /tmp is a general scratch area and must be writable. Node/Bun apps
	// extract native C++ extensions (.node/.so) here, so 'exec' is required.
	// /run is used by some runtimes for PID files etc.
	// /root is $HOME for root-running processes (agents like claude, codex).
	// /home provides writable home dirs for any non-root user.
	// /usr/local/share/ca-certificates is used by Docker Desktop/Orbstack to inject custom CAs.
	tmpfs := map[string]string{
		"/tmp":                             "mode=1777,size=256m,exec",
		"/run":                             "mode=0755,size=64m,exec",
		"/root":                            "mode=0700,size=256m,exec",
		"/home":                            "mode=0755,size=256m,exec",
		"/usr/local/share/ca-certificates": "mode=0755,size=64m",
	}

	// --- User ---
	user := cfg.UserMapping

	return SecurityOptions{
		CapDrop:        capDrop,
		SecurityOpt:    []string{seccompOpt},
		ReadonlyRootfs: cfg.ReadOnlyRoot,
		Tmpfs:          tmpfs,
		Resources:      resources,
		User:           user,
	}, nil
}
