// Package config provides configuration loading and management for the sandbox CLI.
package config

// Config is the top-level configuration structure for the sandbox tool.
type Config struct {
	// Images maps agent names to their Docker base images.
	Images map[string]string `mapstructure:"images" yaml:"images"`

	// EnvWhitelist is the list of env var names (with glob support) safe to
	// forward from the host into the container.
	EnvWhitelist []string `mapstructure:"env_whitelist" yaml:"env_whitelist"`

	// EnvBlocklist is the list of env var names (with glob support) that must
	// never be forwarded into the container, even if whitelisted.
	EnvBlocklist []string `mapstructure:"env_blocklist" yaml:"env_blocklist"`

	// Container holds container runtime settings.
	Container ContainerConfig `mapstructure:"container" yaml:"container"`

	// Security holds resource limit and confinement settings.
	Security SecurityConfig `mapstructure:"security" yaml:"security"`

	// Logging holds log level and format settings.
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging"`

	// Paths holds filesystem path configuration.
	Paths PathConfig `mapstructure:"paths" yaml:"paths"`
}

// ContainerConfig holds container runtime defaults.
type ContainerConfig struct {
	// Timeout is the maximum duration a container may run (e.g. "30m").
	Timeout string `mapstructure:"timeout" yaml:"timeout"`

	// NetworkMode is the Docker network mode (e.g. "bridge").
	NetworkMode string `mapstructure:"network_mode" yaml:"network_mode"`

	// Remove indicates whether the container should be removed on exit.
	Remove bool `mapstructure:"remove" yaml:"remove"`
}

// SecurityConfig holds resource limits and isolation settings for containers.
type SecurityConfig struct {
	// MemoryLimit is the maximum memory a container may use (e.g. "4GB", "512MB").
	// Zero or empty string means no limit.
	MemoryLimit string `mapstructure:"memory_limit" yaml:"memory_limit"`

	// CPUQuota is the CPU quota in microseconds per 100ms period.
	// 0 means unlimited (default).
	CPUQuota int64 `mapstructure:"cpu_quota" yaml:"cpu_quota"`

	// PidsLimit is the maximum number of PIDs allowed inside the container.
	// 0 means unlimited; recommended default is 512.
	PidsLimit int64 `mapstructure:"pids_limit" yaml:"pids_limit"`

	// SeccompProfilePath is an optional path to a custom seccomp JSON profile.
	// When empty the built-in default profile is used.
	SeccompProfilePath string `mapstructure:"seccomp_profile_path" yaml:"seccomp_profile_path"`

	// ReadOnlyRoot mounts the container rootfs as read-only when true (default).
	ReadOnlyRoot bool `mapstructure:"read_only_root" yaml:"read_only_root"`

	// UserMapping is the "uid:gid" string used to run the container process as
	// a non-root user. Defaults to "65534:65534" (nobody).
	UserMapping string `mapstructure:"user_mapping" yaml:"user_mapping"`

	// DropCapabilities lists Linux capabilities to drop from the container.
	// Merged with the hardcoded minimum set at runtime.
	DropCapabilities []string `mapstructure:"drop_capabilities" yaml:"drop_capabilities"`
}

// LoggingConfig holds logging preferences.
type LoggingConfig struct {
	// Level is the minimum log level: debug, info, warn, error.
	Level string `mapstructure:"level" yaml:"level"`

	// Format is the output format: "json" or "console".
	Format string `mapstructure:"format" yaml:"format"`
}

// PathConfig holds well-known filesystem paths used by the sandbox.
type PathConfig struct {
	// Workspace is the mount point inside the container (default: "/work").
	Workspace string `mapstructure:"workspace" yaml:"workspace"`

	// ConfigDir is the host-side directory for sandbox configuration.
	ConfigDir string `mapstructure:"config_dir" yaml:"config_dir"`

	// CacheDir is the host-side directory for cached data.
	CacheDir string `mapstructure:"cache_dir" yaml:"cache_dir"`
}
