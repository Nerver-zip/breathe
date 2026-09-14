package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

func NewCleanCommand() *cobra.Command {
	var all bool
	cleanCmd := &cobra.Command{
		Use:     "clean [target]",
		Aliases: []string{"clear", "reset", "delete", "rm"},
		Short:   "Clean or delete recorded breathing sessions",
		Long: `Clean recorded breathing sessions from history.
Operates on session scope. By default cleans the latest session (~0).
Supports git-like offsets (~1 for previous, ~2 for penultimate, etc.) or --all to clear everything.

Examples:
  breath stats clean        # Delete latest session
  breath stats clean ~1     # Delete previous session
  breath stats clean ~2     # Delete penultimate session
  breath stats clean --all  # Delete all sessions`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanSessions(cmd, args, all)
		},
	}
	cleanCmd.Flags().BoolVarP(&all, "all", "a", false, "delete all recorded sessions")
	return cleanCmd
}

func runCleanSessions(cmd *cobra.Command, args []string, all bool) error {
	store, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	if all {
		count, err := store.DeleteAllSessions(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			cmd.Println("No sessions to delete. History is already empty.")
			return nil
		}
		suffix := "s"
		if count == 1 {
			suffix = ""
		}
		cmd.Printf("Cleared %d session%s from history.\n", count, suffix)
		return nil
	}

	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	offset, err := parseSessionOffset(target)
	if err != nil {
		return err
	}

	del, err := store.DeleteSessionByOffset(ctx, offset)
	if err != nil {
		return err
	}

	desc := "latest session"
	if offset == 1 {
		desc = "previous session (~1)"
	} else if offset > 1 {
		desc = fmt.Sprintf("session ~%d", offset)
	}

	timeStr := del.StartedAt.In(time.Local).Format("2006-01-02 15:04")
	roundSuffix := "s"
	if del.CompletedRounds == 1 {
		roundSuffix = ""
	}
	cmd.Printf("Deleted %s (ID: #%d, started: %s, %d round%s, active: %s)\n",
		desc,
		del.ID,
		timeStr,
		del.CompletedRounds,
		roundSuffix,
		storage.FormatDuration(del.ActiveDuration),
	)
	return nil
}

func parseSessionOffset(target string) (int, error) {
	target = strings.TrimSpace(target)
	if target == "" || target == "latest" || target == "last" || target == "HEAD" || target == "~0" || target == "0" {
		return 0, nil
	}

	t := target
	t = strings.TrimPrefix(t, "HEAD")
	t = strings.TrimPrefix(t, "@")
	t = strings.TrimPrefix(t, "~")
	val, err := strconv.Atoi(t)
	if err != nil || val < 0 {
		return 0, fmt.Errorf("invalid session target %q (expected ~1, ~2, ..., or --all)", target)
	}
	return val, nil
}

func init() {
	statsCmd.Flags().BoolVar(&flagPlain, "plain", false, "display statistics in plain text format")
	statsCmd.Flags().BoolVar(&flagJSON, "json", false, "display statistics in JSON format")
	statsCmd.AddCommand(NewCleanCommand())
}
