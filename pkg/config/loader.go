// Package config provides configuration loading and management for the sandbox CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Load reads and returns configuration from ~/.sandbox/config.yaml (if it
// exists), merged with the hardcoded defaults. Errors are returned only for
// structural issues (e.g. malformed YAML); a missing config file is silently
// ignored and defaults are used.
func Load(logger *zap.Logger) (*Config, error) {
	logger.Debug("loading configuration")

	v := viper.New()
	applyDefaults(v)

	// Resolve config directory (~/.sandbox) and configure Viper.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".sandbox")
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Attempt to read the config file; if it doesn't exist just use defaults.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		logger.Debug("config file not found, using defaults",
			zap.String("config_dir", configDir),
		)
	} else {
		logger.Debug("config file loaded",
			zap.String("config_file", v.ConfigFileUsed()),
		)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	logger.Info("configuration loaded",
		zap.String("log_level", cfg.Logging.Level),
		zap.String("log_format", cfg.Logging.Format),
		zap.String("workspace", cfg.Paths.Workspace),
		zap.String("timeout", cfg.Container.Timeout),
	)

	return &cfg, nil
}

// WriteDefault creates a default config file at ~/.sandbox/config.yaml if the
// directory (and file) do not yet exist. This is called on the first run.
func WriteDefault(logger *zap.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".sandbox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		// File already exists; nothing to do.
		return nil
	}

	v := viper.New()
	applyDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal defaults: %w", err)
	}

	yamlData, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	header := []byte("# Sandbox CLI Configuration\n# See https://github.com/servusdei2018/sandbox for documentation.\n\n")
	fullContent := append(header, yamlData...)

	if err := os.WriteFile(configFile, fullContent, 0o644); err != nil {
		return fmt.Errorf("failed to write default config to %s: %w", configFile, err)
	}

	logger.Info("default config file created from defaults", zap.String("path", configFile))
	return nil
}
