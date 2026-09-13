package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show breathing statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := storage.Open("")
		if err != nil {
			return err
		}
		defer store.Close()
		s, err := store.Summary(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("Today: %d sessions • %d rounds • best retention %s\n", s.TodaySessions, s.TodayRounds, durationLabel(s.TodayBestRetention))
		fmt.Printf("All time: %d sessions • %d rounds • active %s\n", s.TotalSessions, s.TotalRounds, durationLabel(s.TotalActive))
		fmt.Printf("Retention: average %s • best %s\n", durationLabel(s.AverageRetention), durationLabel(s.BestRetention))
		return nil
	},
}

func durationLabel(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}
	total := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}
