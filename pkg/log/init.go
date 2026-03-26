// Package log provides a global Zap logger instance for use across all packages.
// Call Init() once at startup with the desired log level and format.
package log

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global logger instance. It is initialized by Init().
var Logger *zap.Logger

// Init initializes the global Logger with the specified level and format.
// level should be one of: "debug", "info", "warn", "error".
// format should be one of: "json" or "console".
func Init(level, format string) error {
	var cfg zap.Config

	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return fmt.Errorf("invalid log level %q: %w", level, err)
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %w", err)
	}

	Logger = logger
	return nil
}

// Noop returns a no-op logger suitable for testing.
func Noop() *zap.Logger {
	return zap.NewNop()
}
