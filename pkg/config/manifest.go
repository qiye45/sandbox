package config

import (
	"errors"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ProjectManifest represents a .sandbox.yml file in a user's repository.
type ProjectManifest struct {
	// Setup is a list of shell commands to run before executing the agent.
	Setup []string `yaml:"setup"`
}

// LoadManifest attempts to load a .sandbox.yml from the specified directory.
// If the file does not exist, it returns an empty manifest and no error.
func LoadManifest(workspaceDir string, logger *zap.Logger) (*ProjectManifest, error) {
	manifestPath := filepath.Join(workspaceDir, ".sandbox.yml")

	bytes, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Debug("no .sandbox.yml manifest found in workspace", zap.String("path", workspaceDir))
			return &ProjectManifest{}, nil
		}
		return nil, err
	}

	logger.Debug("reading project manifest", zap.String("path", manifestPath))

	var manifest ProjectManifest
	if err := yaml.Unmarshal(bytes, &manifest); err != nil {
		return nil, err
	}

	logger.Info("manifest loaded successfully",
		zap.String("path", manifestPath),
		zap.Int("setup_commands", len(manifest.Setup)),
	)

	return &manifest, nil
}
