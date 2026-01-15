package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"summit/pkg/log"
	"summit/pkg/system"

	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	logLevel   string
	jsonOutput bool
	logger     log.Logger
	cmdRunner  system.CommandRunner = &system.LiveCommandRunner{}
	rootCmd                         = &cobra.Command{
		Use:   "summit",
		Short: "summit is a tool for managing Alpine Linux installations",
		Long: `A declarative tool for managing all aspects of an Alpine Linux installation,
from package installs to system configs and services enablement and startup.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := parseLogLevel(logLevel)
			if err != nil {
				return err
			}
			writer := cmd.ErrOrStderr()
			logger = log.NewSlogLogger(level, writer)
			ctx := context.WithValue(cmd.Context(), "logger", logger)
			cmd.SetContext(ctx)
			return nil
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseLogLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", levelStr)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./system.yaml", "config file (default is ./system.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
}
