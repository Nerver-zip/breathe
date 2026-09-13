package cmd

import (
	"fmt"

	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}
		path, _ := appconfig.Path()
		fmt.Printf("path: %s\nrounds: %d\nbreathing: %s\nrecovery: %s\nauto_next_round: %t\ntheme: %s\nnotifications: %t\n",
			path, cfg.Rounds, cfg.Breathing, cfg.Recovery, cfg.AutoNextRound, cfg.Theme, cfg.Notifications)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print configuration path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := appconfig.Path()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd, configPathCmd)
}
