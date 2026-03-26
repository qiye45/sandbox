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
