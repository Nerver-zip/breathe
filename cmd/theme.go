package cmd

import (
	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage and preview color themes",
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all built-in themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}
		active := cfg.Theme
		cmd.Println("Available themes:")
		for _, th := range theme.List() {
			marker := "  "
			if th.Name == active {
				marker = "* "
			}
			cmd.Printf("%s%-18s %s\n", marker, th.Name, th.Description)
		}
		cmd.Println("\nRun 'breath theme preview <name>' to preview a theme.")
		return nil
	},
}

var themeSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set active theme",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := appconfig.Set("theme", name)
		if err != nil {
			return err
		}
		cmd.Printf("Theme changed to '%s'\n", cfg.Theme)
		return nil
	},
}

var themePreviewCmd = &cobra.Command{
	Use:   "preview [name]",
	Short: "Preview a theme's visual styling",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			cfg, err := appconfig.Load()
			if err != nil {
				return err
			}
			name = cfg.Theme
		}
		out, err := theme.Preview(name)
		if err != nil {
			return err
		}
		cmd.Println(out)
		return nil
	},
}

func init() {
	themeCmd.AddCommand(themeListCmd, themeSetCmd, themePreviewCmd)
}
