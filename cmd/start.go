package cmd

import (
	"context"
	"fmt"
	"time"

	appconfig "github.com/Nerver-zip/breathing-tui/config"
	"github.com/Nerver-zip/breathing-tui/internal/notify"
	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
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
	for _, c := range []*cobra.Command{startCmd, rootCmd} {
		c.Flags().IntVarP(&flagRounds, "rounds", "r", defaults.Rounds, "number of rounds")
		c.Flags().DurationVar(&flagBreathing, "breathing", defaults.Breathing, "breathing phase duration")
		c.Flags().DurationVar(&flagRecovery, "recovery", defaults.Recovery, "recovery hold duration")
		c.Flags().BoolVar(&flagAutoNext, "auto-next", defaults.AutoNextRound, "automatically start the next round after recovery")
		c.Flags().StringVar(&flagTheme, "theme", "", "theme override")
	}
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
	themeName := cfg.Theme

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
			themeName = flagTheme
		}
	}
	if rounds < 1 || breathing <= 0 || recovery <= 0 {
		return fmt.Errorf("rounds and durations must be positive")
	}
	if _, err := theme.Get(themeName); err != nil {
		return err
	}

	store, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer store.Close()

	ctx := context.Background()
	_ = store.CleanupUnfinishedSessions(ctx)

	startedAt := time.Now()
	sessionID, err := store.CreateSession(ctx, rounds, startedAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	notif := notify.New(notify.Options{
		Desktop: cfg.Notifications,
		Bell:    cfg.Bell,
	})

	engine := session.New(session.Settings{
		Rounds:        rounds,
		Breathing:     breathing,
		Recovery:      recovery,
		AutoNextRound: autoNext,
	})

	model := tui.NewSessionModel(engine, store, sessionID, startedAt, notif, themeName)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	fm, ok := finalModel.(tui.Model)
	if !ok {
		return nil
	}

	status := "abandoned"
	if fm.Engine().Done() {
		status = "completed"
	}
	_ = store.EndSession(ctx, sessionID, time.Now(), fm.Engine().SessionElapsed(), time.Since(startedAt), status)

	if fm.ShouldOpenStats() {
		statsModel := tui.NewStatsModel(store, themeName)
		statsProg := tea.NewProgram(statsModel, tea.WithAltScreen())
		_, _ = statsProg.Run()
	}

	return nil
}
