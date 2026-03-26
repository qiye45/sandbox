package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These variables are injected at build time via -ldflags.
var (
	// Version is the semantic version of the binary (e.g. "0.1.0").
	Version = "dev"
	// Commit is the short git commit hash (e.g. "abc1234").
	Commit = "none"
	// BuildDate is the ISO-8601 build timestamp.
	BuildDate = "unknown"
)

// versionCmd returns a cobra.Command that prints version information.
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("sandbox %s (commit: %s, built: %s, %s/%s)\n",
				Version, Commit, BuildDate, runtime.GOOS, runtime.GOARCH,
			)
			return nil
		},
	}
}
