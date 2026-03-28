package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/cache"
	"github.com/servusdei2018/sandbox/pkg/config"
	sandboxlog "github.com/servusdei2018/sandbox/pkg/log"
)

// cacheCmd returns the "sandbox cache" command tree.
func cacheCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "cache",
		Short: "Manage the global dependency cache",
	}

	parent.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Show details about the global dependency cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cacheDir := cfg.Paths.CacheDir
			if cacheDir == "" {
				return fmt.Errorf("cache_dir is not configured in paths")
			}

			size, files, err := cache.GetStats(cacheDir)
			if err != nil {
				return err
			}

			fmt.Printf("Cache Location : %s\n", cacheDir)
			fmt.Printf("Total Size     : %s\n", cache.FormatSize(size))
			fmt.Printf("Total Files    : %d\n", files)
			return nil
		},
	})

	parent.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Remove all files from the global dependency cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cacheDir := cfg.Paths.CacheDir
			if cacheDir == "" {
				return fmt.Errorf("cache_dir is not configured in paths")
			}

			// Verify the directory exists before cleaning to avoid non-existent path errors.
			if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
				fmt.Printf("Cache is already empty (directory %s does not exist).\n", cacheDir)
				return nil
			}

			if err := cache.Clean(cacheDir); err != nil {
				return err
			}

			logger.Info("cache cleaned successfully", zap.String("path", cacheDir))
			fmt.Printf("Successfully cleaned the cache directory: %s\n", cacheDir)
			return nil
		},
	})

	return parent
}
