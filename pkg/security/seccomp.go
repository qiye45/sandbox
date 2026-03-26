// Package security provides Linux security primitives for the sandbox runtime.
// These include seccomp profiles, cgroup resource limits, and capability management.
package security

import (
	"encoding/json"
	"fmt"
	"os"
)

// seccompProfile mirrors the structure expected by Docker's seccomp JSON format
// (opencontainers/runtime-spec Seccomp type).
type seccompProfile struct {
	DefaultAction string        `json:"defaultAction"`
	Architectures []string      `json:"architectures"`
	Syscalls      []seccompRule `json:"syscalls"`
}

type seccompRule struct {
	Names  []string `json:"names"`
	Action string   `json:"action"`
	// ErrnoRet is the errno value returned to the process when the syscall is
	// blocked. Must be set explicitly; Docker defaults to 0 (no error) if
	// omitted, which makes the syscall appear to succeed with a garbage result.
	ErrnoRet uint `json:"errnoRet"`
	// Comment is ignored at runtime but aids human review.
	Comment string `json:"comment,omitempty"`
}

// defaultDeniedSyscalls is the set of syscalls we block from sandboxed processes.
//
// We use a deny-list (defaultAction=SCMP_ACT_ALLOW) so legitimate workloads keep
// working out of the box, while the most dangerous primitives are blocked.
var defaultDeniedSyscalls = []seccompRule{
	{
		Names: []string{
			"mount", "umount", "umount2",
			"pivot_root", "chroot",
		},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1, // EPERM
		Comment:  "prevent filesystem namespace escape",
	},
	{
		Names:    []string{"ptrace"},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1,
		Comment:  "prevent process tracing / memory inspection",
	},
	{
		Names: []string{
			"kexec_load", "kexec_file_load",
		},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1,
		Comment:  "prevent loading a new kernel; bpf/perf_event_open omitted — already blocked by dropped caps",
	},
	{
		Names: []string{
			"init_module", "finit_module", "delete_module",
		},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1,
		Comment:  "prevent kernel module loading/unloading",
	},
	{
		Names: []string{
			"reboot",
			"adjtimex", "clock_adjtime", "ntp_adjtime",
		},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1,
		Comment:  "prevent administrative kernel operations",
	},
	{
		Names: []string{
			"unshare",
			"setns",
		},
		Action:   "SCMP_ACT_ERRNO",
		ErrnoRet: 1,
		Comment:  "prevent namespace manipulation",
	},
}

// defaultProfile returns the built-in seccomp profile used by the sandbox.
func defaultProfile() *seccompProfile {
	return &seccompProfile{
		DefaultAction: "SCMP_ACT_ALLOW",
		Architectures: []string{
			"SCMP_ARCH_X86_64",
			"SCMP_ARCH_X86",
			"SCMP_ARCH_AARCH64",
		},
		Syscalls: defaultDeniedSyscalls,
	}
}

// DefaultProfileJSON serialises the built-in seccomp profile to JSON.]
//
// The result is suitable for passing directly to Docker's security-opt
// seccomp=<inline-json> or writing to a file.
func DefaultProfileJSON() ([]byte, error) {
	b, err := json.MarshalIndent(defaultProfile(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("seccomp: failed to marshal default profile: %w", err)
	}
	return b, nil
}

// LoadOrDefault returns the raw JSON of a seccomp profile.
//
// If profilePath is non-empty the file is read; otherwise DefaultProfileJSON
// is returned.
func LoadOrDefault(profilePath string) ([]byte, error) {
	if profilePath == "" {
		return DefaultProfileJSON()
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("seccomp: failed to read profile %q: %w", profilePath, err)
	}
	return data, nil
}
