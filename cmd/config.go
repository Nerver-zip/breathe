package cmd

import (
	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and modify configuration",
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
		cmd.Printf("path: %s\nrounds: %d\nbreathing: %s\nrecovery: %s\nauto_next_round: %t\ntheme: %s\nnotifications: %t\nbell: %t\n",
			path, cfg.Rounds, cfg.Breathing, cfg.Recovery, cfg.AutoNextRound, cfg.Theme, cfg.Notifications, cfg.Bell)
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
		cmd.Println(path)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := args[1]
		cfg, err := appconfig.Set(key, val)
		if err != nil {
			return err
		}
		cmd.Printf("Updated %s to %s\n", key, val)
		path, _ := appconfig.Path()
		cmd.Printf("Config saved to %s (effective theme: %s, rounds: %d, breathing: %s, recovery: %s)\n",
			path, cfg.Theme, cfg.Rounds, cfg.Breathing, cfg.Recovery)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd, configPathCmd, configSetCmd)
}
