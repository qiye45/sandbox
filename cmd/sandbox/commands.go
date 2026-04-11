package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/agent"
	"github.com/servusdei2018/sandbox/pkg/config"
	cnt "github.com/servusdei2018/sandbox/pkg/container"
	sandboxlog "github.com/servusdei2018/sandbox/pkg/log"
)

// ExitError is a custom error type that carries an exit code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// runCmd returns the "sandbox run <agent> [args...]" subcommand.
func runCmd() *cobra.Command {
	var (
		imageFlagOverride string
		timeoutFlag       string
		workspaceFlag     string
		keepContainer     bool
		seccompFlag       string
		pruneUnusedFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "run <binary> [args...]",
		Short: "Run a command or agent in a sandbox container",
		Long: `Run executes the given binary (and optional arguments) inside an isolated
Docker container, with the current working directory bind-mounted to /work.

Examples:
  sandbox run echo hello
  sandbox run python -c "print('hello')"
  sandbox run claude --help`,
		Args: cobra.MinimumNArgs(1),
		// DisableFlagParsing would break our own flags (--image, --timeout, etc.).
		// Instead, SetInterspersed(false) below stops pflag from consuming agent
		// flags like -c, -e, --help that appear after the first positional arg.
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			logger.Info("sandbox run requested",
				zap.Strings("args", args),
			)

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if cfgWriteErr := config.WriteDefault(logger); cfgWriteErr != nil {
				logger.Warn("could not write default config file", zap.Error(cfgWriteErr))
			}

			// Apply config-level image overrides to the agent registry.
			agent.OverrideImages(cfg.Images)

			hostEnv := agent.HostEnv()
			detectedAgent, err := agent.Detect(args, hostEnv, cfg.Images, logger)
			if err != nil {
				return fmt.Errorf("agent detection failed: %w", err)
			}

			// Determine final image: CLI flag wins, then detected agent image.
			finalImage := detectedAgent.Image
			if imageFlagOverride != "" {
				finalImage = imageFlagOverride
				logger.Info("image overridden by flag", zap.String("image", finalImage))
			}

			rawTimeout := cfg.Container.Timeout
			if timeoutFlag != "" {
				rawTimeout = timeoutFlag
			}
			timeout, err := time.ParseDuration(rawTimeout)
			if err != nil {
				return fmt.Errorf("invalid timeout %q: %w", rawTimeout, err)
			}
			logger.Info("parsed timeout", zap.Duration("duration", timeout))

			wsDir := workspaceFlag
			if wsDir == "" {
				wsDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine current directory: %w", err)
				}
			}

			// Resolve as absolute path and handle symlinks so Docker bind-mount
			// works on all systems.
			absWsDir, err := filepath.Abs(wsDir)
			if err != nil {
				return fmt.Errorf("failed to resolve absolute path for %s: %w", wsDir, err)
			}
			wsDir, err = filepath.EvalSymlinks(absWsDir)
			if err != nil {
				return fmt.Errorf("failed to resolve workspace path %s: %w", wsDir, err)
			}
			logger.Debug("resolved workspace directory", zap.String("path", wsDir))

			manifest, err := config.LoadManifest(wsDir, logger)
			if err != nil {
				// Don't error out hard; falling back to empty manifest.
				logger.Warn("could not load project manifest", zap.Error(err))
				manifest = &config.ProjectManifest{}
			}

			filteredEnv := cnt.FilterEnv(hostEnv, cfg.EnvWhitelist, cfg.EnvBlocklist, logger)
			filteredEnv = cnt.ApplyEnvSpecs(filteredEnv, hostEnv, cfg.Env, logger)

			extraBinds := make([]string, 0, len(cfg.Paths.MountTargets))
			for _, mountTarget := range cfg.Paths.MountTargets {
				hostPath := strings.TrimSpace(mountTarget.Source)
				targetPath := strings.TrimSpace(mountTarget.Target)
				mode := config.NormalizeMountMode(mountTarget.Mode)

				if hostPath == "" || targetPath == "" {
					logger.Warn("skipping invalid mount target entry", zap.String("host", hostPath), zap.String("target", targetPath))
					continue
				}
				if !strings.HasPrefix(targetPath, "/") {
					logger.Warn("skipping mount target with non-absolute container path", zap.String("host", hostPath), zap.String("target", targetPath))
					continue
				}

				resolvedHostPath, err := filepath.Abs(hostPath)
				if err != nil {
					logger.Warn("skipping mount target with invalid host path", zap.String("host", hostPath), zap.Error(err))
					continue
				}
				resolvedHostPath = filepath.Clean(resolvedHostPath)

				if resolvedPath, err := filepath.EvalSymlinks(resolvedHostPath); err == nil {
					resolvedHostPath = resolvedPath
				}

				if _, err := os.Stat(resolvedHostPath); err != nil {
					logger.Warn("skipping mount target because host path does not exist", zap.String("host", resolvedHostPath), zap.Error(err))
					continue
				}

				if strings.TrimSpace(mountTarget.Mode) != "" && mode != strings.ToLower(strings.TrimSpace(mountTarget.Mode)) {
					logger.Warn("invalid mount mode, defaulting to write mode",
						zap.String("host", resolvedHostPath),
						zap.String("target", targetPath),
						zap.String("mode", mountTarget.Mode),
					)
				}

				bindSpec := fmt.Sprintf("%s:%s", resolvedHostPath, targetPath)
				if mode == config.MountModeRead {
					bindSpec += ":ro"
				}
				extraBinds = append(extraBinds, bindSpec)
			}

			gitMaskBind, gitMaskCleanup, err := workspaceGitMaskBind(
				wsDir,
				cfg.Paths.Workspace,
				filepath.Join(cfg.Paths.ConfigDir, "tmp"),
			)
			if err != nil {
				return fmt.Errorf("failed to prepare workspace git exclusion: %w", err)
			}
			defer gitMaskCleanup()
			if gitMaskBind != "" {
				extraBinds = append(extraBinds, gitMaskBind)
			}

			hostPathBinds, mountedPathEntries := hostPathReadonlyBinds(os.Getenv("PATH"), logger)
			extraBinds = append(extraBinds, hostPathBinds...)

			manager, err := cnt.NewManager(logger)
			if err != nil {
				return fmt.Errorf("could not connect to Docker daemon: %w", err)
			}
			defer func() {
				if err := manager.Close(); err != nil {
					logger.Warn("failed to close docker manager", zap.Error(err))
				}
			}()

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if cfg.Container.PruneUnusedBeforeRun || pruneUnusedFlag {
				if err := manager.Prune(ctx); err != nil {
					logger.Warn("failed to prune unused sandbox containers before run", zap.Error(err))
				}
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := manager.PullIfMissing(ctx, finalImage); err != nil {
				return fmt.Errorf("image pull failed: %w", err)
			}

			var imageEntrypoint []string
			var imageCmd []string
			var imageEnv []string
			imgInfo, inspectErr := manager.InspectImage(ctx, finalImage)
			if inspectErr != nil {
				logger.Warn("failed to inspect image for defaults", zap.String("image", finalImage), zap.Error(inspectErr))
			} else if imgInfo.Config != nil {
				imageEntrypoint = append([]string{}, imgInfo.Config.Entrypoint...)
				imageCmd = append([]string{}, imgInfo.Config.Cmd...)
				imageEnv = append([]string{}, imgInfo.Config.Env...)
			}

			if len(mountedPathEntries) > 0 {
				filteredEnv = cnt.UpsertEnvValue(
					filteredEnv,
					"PATH",
					strings.Join(mountedPathEntries, string(os.PathListSeparator)),
				)
			}

			filteredEnv = cnt.MergePathWithImageEnv(filteredEnv, imageEnv, logger)

			containerCmd := args
			var entrypoint []string
			// If an entrypoint is defined for this agent, use it.
			if len(detectedAgent.Entrypoint) > 0 {
				entrypoint = append([]string{}, detectedAgent.Entrypoint...)

				// Strip the binary name from args if it matches the agent's expected binary.
				// This allows "sandbox run claude --help" to map correctly
				// when "claude" is the entrypoint.
				if len(args) > 0 && detectedAgent.Name != agent.TypeGeneric {
					containerCmd = args[1:]
				}
			}

			var wrapperMount string
			if len(manifest.Setup) > 0 {
				logger.Info("generating wrapper script for setup commands", zap.Int("count", len(manifest.Setup)))

				if len(entrypoint) == 0 {
					entrypoint = append([]string{}, imageEntrypoint...)
					if len(entrypoint) == 0 && len(containerCmd) == 0 {
						containerCmd = append([]string{}, imageCmd...)
					}
				}

				fullCmd := append(entrypoint, containerCmd...)

				tmpDir := filepath.Join(cfg.Paths.ConfigDir, "tmp")
				if err := os.MkdirAll(tmpDir, 0o700); err != nil {
					return fmt.Errorf("failed to create tmp dir for wrapper script: %w", err)
				}

				wrapperFile, err := os.CreateTemp(tmpDir, "wrapper-*.sh")
				if err != nil {
					return fmt.Errorf("failed to create wrapper script: %w", err)
				}

				wrapperPath := wrapperFile.Name()
				defer func() {
					if err := os.Remove(wrapperPath); err != nil {
						logger.Warn("failed to remove wrapper script", zap.String("path", wrapperPath), zap.Error(err))
					}
				}()

				scriptContent := "#!/bin/sh\nset -e\n"
				for _, step := range manifest.Setup {
					scriptContent += fmt.Sprintf("echo '=== Running setup: %s ==='\n", step)
					scriptContent += step + "\n"
				}

				scriptContent += "echo '=== Setup complete. Executing main process. ==='\nexec"
				for _, c := range fullCmd {
					scriptContent += " " + fmt.Sprintf("%q", c)
				}
				scriptContent += "\n"

				if _, err := wrapperFile.Write([]byte(scriptContent)); err != nil {
					_ = wrapperFile.Close()
					return fmt.Errorf("failed to write wrapper script: %w", err)
				}
				if err := wrapperFile.Close(); err != nil {
					return fmt.Errorf("failed to close wrapper script: %w", err)
				}

				if err := os.Chmod(wrapperPath, 0o755); err != nil {
					return fmt.Errorf("failed to chmod wrapper script: %w", err)
				}

				// Ensure the host path is absolute for Docker.
				absWrapperPath, err := filepath.Abs(wrapperPath)
				if err != nil {
					return fmt.Errorf("failed to resolve absolute path for wrapper script: %w", err)
				}

				logger.Debug("generated entrypoint wrapper", zap.String("path", absWrapperPath))

				entrypoint = []string{"/sandbox-entrypoint.sh"}
				containerCmd = nil
				wrapperMount = fmt.Sprintf("%s:/sandbox-entrypoint.sh:ro", absWrapperPath)
			}

			containerCfg := &cnt.Config{
				Image:        finalImage,
				Cmd:          containerCmd,
				Entrypoint:   entrypoint,
				WorkspaceDir: wsDir,
				MountTarget:  cfg.Paths.Workspace,
				Env:          filteredEnv,
				Timeout:      timeout,
				RemoveOnExit: cfg.Container.Remove && !keepContainer,
				NetworkMode:  cfg.Container.NetworkMode,
				Tty:          true,
				AttachStdin:  true,
				Security:     cfg.Security,
				CacheDir:     cfg.Paths.CacheDir,
				ExtraBinds:   extraBinds,
				WrapperMount: wrapperMount,
			}

			if seccompFlag != "" {
				containerCfg.Security.SeccompProfilePath = seccompFlag
			}

			containerID, err := manager.Create(ctx, containerCfg)
			if err != nil {
				return fmt.Errorf("sandbox setup failed: %w", err)
			}

			// We use signal.NotifyContext above to handle graceful shutdown.
			// The context will be canceled on SIGINT/SIGTERM, which will cause
			// manager.Run to return (as it blocks on ContainerWait which
			// respects context cancellation).

			exitCode, err := manager.Run(ctx, containerID, containerCfg.Tty)
			if err != nil {
				// If the context was canceled, it's likely a signal or timeout.
				if ctx.Err() != nil {
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						logger.Warn("container execution timed out",
							zap.Duration("timeout", timeout),
						)
					} else {
						logger.Info("container execution interrupted", zap.Error(ctx.Err()))
					}

					// Try to stop the container before returning.
					stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer stopCancel()
					_ = manager.Stop(stopCtx, containerID)
					return &ExitError{Code: 130}
				}

				logger.Error("container execution failed",
					zap.String("agent", string(detectedAgent.Name)),
					zap.String("image", finalImage),
					zap.Error(err),
				)
				return fmt.Errorf("sandbox execution failed: %w", err)
			}

			// Cleanup: remove container unless --keep was set.
			if containerCfg.RemoveOnExit {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				if err := manager.Remove(cleanupCtx, containerID); err != nil {
					logger.Warn("failed to remove container",
						zap.String("container_id", containerID[:12]),
						zap.Error(err),
					)
				}
			}

			if exitCode != 0 {
				return &ExitError{Code: exitCode}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&imageFlagOverride, "image", "i", "", "Docker image to use (overrides auto-detection)")
	cmd.Flags().StringVarP(&timeoutFlag, "timeout", "t", "", "Maximum execution time (e.g. 30m, 1h)")
	cmd.Flags().StringVarP(&workspaceFlag, "workspace", "w", "", "Host directory to mount as /work (defaults to cwd)")
	cmd.Flags().BoolVarP(&keepContainer, "keep", "k", false, "Do not remove the container after execution")
	cmd.Flags().StringVar(&seccompFlag, "seccomp", "", "Path to a custom seccomp JSON profile")
	cmd.Flags().BoolVar(&pruneUnusedFlag, "prune-unused", false, "Remove stopped sandbox containers before execution")

	// Stop flag parsing at the first non-flag argument so agent flags
	// (e.g. python -c, node -e, claude --help) are not consumed by Cobra.
	cmd.Flags().SetInterspersed(false)

	return cmd
}

// configCmd returns the "sandbox config show" subcommand.
func configCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "config",
		Short: "Manage sandbox configuration",
	}

	parent.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Workspace mount point : %s\n", cfg.Paths.Workspace)
			fmt.Printf("Extra mounts          : %d\n", len(cfg.Paths.MountTargets))
			fmt.Printf("Config directory      : %s\n", cfg.Paths.ConfigDir)
			fmt.Printf("Cache directory       : %s\n", cfg.Paths.CacheDir)
			fmt.Printf("Container timeout     : %s\n", cfg.Container.Timeout)
			fmt.Printf("Network mode          : %s\n", cfg.Container.NetworkMode)
			fmt.Printf("Remove on exit        : %v\n", cfg.Container.Remove)
			fmt.Printf("Prune unused before run : %v\n", cfg.Container.PruneUnusedBeforeRun)
			fmt.Printf("Configured env entries: %d\n", len(cfg.Env))
			fmt.Printf("Log level             : %s\n", cfg.Logging.Level)
			fmt.Printf("Log format            : %s\n", cfg.Logging.Format)
			fmt.Printf("Default image         : %s\n", cfg.Images["default"])
			return nil
		},
	})

	parent.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create the default config file at ~/.sandbox/config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}
			return config.WriteDefault(logger)
		},
	})

	return parent
}

// pruneCmd returns the "sandbox prune" subcommand.
func pruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove stopped sandbox containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			manager, err := cnt.NewManager(logger)
			if err != nil {
				return fmt.Errorf("could not connect to Docker daemon: %w", err)
			}
			defer func() {
				if err := manager.Close(); err != nil {
					logger.Warn("failed to close docker manager", zap.Error(err))
				}
			}()

			return manager.Prune(context.Background())
		},
	}
}
