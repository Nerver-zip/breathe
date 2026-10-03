package cmd

import (
	"fmt"
	"os"
	"strings"

	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/Nerver-zip/breathing-tui/internal/timezone"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "breath",
	Short:   "A terminal breathing-session timer and tracker",
	Version: version,
	Long: `Breathing TUI guides timed breathing rounds, open-ended breath retention,
and recovery holds while recording completed sessions locally.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err == nil && cfg.Timezone != "" && !strings.EqualFold(cfg.Timezone, "auto") {
			_, _ = timezone.Set(cfg.Timezone)
		} else {
			_, _ = timezone.DetectAndSet("")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStart(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(themeCmd)
	rootCmd.AddCommand(NewCleanCommand())
}
