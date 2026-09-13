package cmd

import (
	"context"
	"fmt"
	"time"

	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a guided breathing session",
	RunE:  runStart,
}

var (
	flagRounds    int
	flagBreathing time.Duration
	flagRecovery  time.Duration
	flagAutoNext  bool
	flagTheme     string
)

func init() {
	defaults := appconfig.Defaults()
	startCmd.Flags().IntVarP(&flagRounds, "rounds", "r", defaults.Rounds, "number of rounds")
	startCmd.Flags().DurationVar(&flagBreathing, "breathing", defaults.Breathing, "breathing phase duration")
	startCmd.Flags().DurationVar(&flagRecovery, "recovery", defaults.Recovery, "recovery hold duration")
	startCmd.Flags().BoolVar(&flagAutoNext, "auto-next", defaults.AutoNextRound, "automatically start the next round after recovery")
	startCmd.Flags().StringVar(&flagTheme, "theme", "", "theme override")
}

func runStart(cmd *cobra.Command, _ []string) error {
	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}

	rounds := cfg.Rounds
	breathing := cfg.Breathing
	recovery := cfg.Recovery
	autoNext := cfg.AutoNextRound
	theme := cfg.Theme

	if cmd != nil {
		if cmd.Flags().Changed("rounds") {
			rounds = flagRounds
		}
		if cmd.Flags().Changed("breathing") {
			breathing = flagBreathing
		}
		if cmd.Flags().Changed("recovery") {
			recovery = flagRecovery
		}
		if cmd.Flags().Changed("auto-next") {
			autoNext = flagAutoNext
		}
		if cmd.Flags().Changed("theme") && flagTheme != "" {
			theme = flagTheme
		}
	}
	if rounds < 1 || breathing <= 0 || recovery <= 0 {
		return fmt.Errorf("rounds and durations must be positive")
	}

	engine := session.New(session.Settings{
		Rounds:        rounds,
		Breathing:     breathing,
		Recovery:      recovery,
		AutoNextRound: autoNext,
	})
	model := tui.New(engine, theme)
	startedAt := time.Now()
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	endedAt := time.Now()
	if err != nil {
		return err
	}

	fm, ok := finalModel.(tui.Model)
	if !ok {
		return fmt.Errorf("unexpected final TUI model")
	}
	status := "abandoned"
	if fm.Engine().Done() {
		status = "completed"
	}
	store, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer store.Close()
	if err := store.SaveSession(context.Background(), storage.SessionRecord{
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		PlannedRounds:  rounds,
		ActiveDuration: fm.Engine().SessionElapsed(),
		Status:         status,
		Rounds:         fm.Engine().Results(),
	}); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}
