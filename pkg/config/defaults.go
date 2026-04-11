package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

// applyDefaults registers all hardcoded default values with the Viper instance.
//
// These values are the lowest priority and will be overridden by the config
// file or any CLI flags.
func applyDefaults(v *viper.Viper, homeDir string) {
	v.SetDefault("images.default", "alpine:latest")
	v.SetDefault("images.claude", "ghcr.io/servusdei2018/sandbox-claude:latest")
	v.SetDefault("images.gemini", "ghcr.io/servusdei2018/sandbox-gemini:latest")
	v.SetDefault("images.codex", "ghcr.io/servusdei2018/sandbox-codex:latest")
	v.SetDefault("images.kilocode", "ghcr.io/servusdei2018/sandbox-kilocode:latest")
	v.SetDefault("images.opencode", "ghcr.io/servusdei2018/sandbox-opencode:latest")
	v.SetDefault("images.python", "python:3.13-alpine")
	v.SetDefault("images.node", "node:24-alpine")
	v.SetDefault("images.bun", "oven/bun:alpine")
	v.SetDefault("images.go", "golang:1.26-alpine")

	// Environment variable whitelist: pass these from host if they exist.
	v.SetDefault("env_whitelist", []string{
		"PATH",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"SHELL",
		"TERM",
		"COLORTERM",
		"XTERM_VERSION",
		"TZ",
	})

	// Environment variable blocklist: never pass these regardless of whitelist.
	v.SetDefault("env_blocklist", []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_*",
		"GCP_*",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GITHUB_TOKEN",
		"GIT_PASSWORD",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"COHERE_API_KEY",
	})

	// Container defaults.
	v.SetDefault("container.timeout", "30m")
	v.SetDefault("container.network_mode", "bridge")
	v.SetDefault("container.remove", true)

	// Security defaults.
	v.SetDefault("security.memory_limit", "4GB")
	v.SetDefault("security.cpu_quota", 0) // 0 = unlimited
	v.SetDefault("security.pids_limit", 512)
	v.SetDefault("security.seccomp_profile_path", "") // empty = built-in default
	v.SetDefault("security.read_only_root", true)
	v.SetDefault("security.user_mapping", "")              // empty = root inside container
	v.SetDefault("security.drop_capabilities", []string{}) // baseline always dropped in code

	// Logging defaults.
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	// Path defaults.
	v.SetDefault("paths.workspace", "/work")
	v.SetDefault("paths.config_dir", filepath.Join(homeDir, ".sandbox"))
	v.SetDefault("paths.cache_dir", filepath.Join(homeDir, ".sandbox", "cache"))
	v.SetDefault("paths.mount_targets", []MountTarget{})
}
