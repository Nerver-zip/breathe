package cmd

import (
	"context"
	"time"

	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flagPlain bool
	flagJSON  bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show breathing practice statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}

		store, err := storage.Open("")
		if err != nil {
			return err
		}
		defer store.Close()

		report, err := store.GetStats(context.Background(), time.Now())
		if err != nil {
			return err
		}

		if flagJSON {
			jsonStr, err := report.JSON()
			if err != nil {
				return err
			}
			cmd.Println(jsonStr)
			return nil
		}

		if flagPlain {
			cmd.Print(report.PlainText())
			return nil
		}

		// Run TUI stats dashboard
		model := tui.NewStatsModel(store, cfg.Theme)
		program := tea.NewProgram(model, tea.WithAltScreen())
		_, err = program.Run()
		return err
	},
}

func init() {
	statsCmd.Flags().BoolVar(&flagPlain, "plain", false, "display statistics in plain text format")
	statsCmd.Flags().BoolVar(&flagJSON, "json", false, "display statistics in JSON format")
}
