package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "breath",
	Short: "A terminal breathing-session timer and tracker",
	Long: `Breathing TUI guides timed breathing rounds, open-ended breath retention,
and recovery holds while recording completed sessions locally.`,
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
}
