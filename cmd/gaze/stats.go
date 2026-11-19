package main

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/masnathan/gaze/internal/stats"
)

func printStats(w io.Writer, name string, m stats.Metrics) error {
	// ponytail: Fixed-width bars fit ordinary terminals; adapt width if narrow terminals become a requirement.
	const width = 20
	bar, percent := strings.Repeat("░", width), "—"
	if m.InputTokens > 0 {
		saved := (1 - float64(m.OutputTokens)/float64(m.InputTokens)) * 100
		filled := int(max(0, min(100, saved)) * width / 100)
		bar = strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		percent = fmt.Sprintf("%.1f%%", saved)
	}
	_, err := fmt.Fprintf(w, "\n%s\n  Tokens saved  %s  %s\n  Tokens  %s → %s  (%s saved)\n  Scans  %d (%d failed)   Files  %d (%d failed)   Time  %s\n", name, bar, percent, compactTokens(m.InputTokens), compactTokens(m.OutputTokens), compactTokens(m.InputTokens-m.OutputTokens), m.Scans, m.FailedScans, m.Files, m.FailedFiles, time.Duration(m.DurationNS))
	return err
}

func compactTokens(n int64) string {
	value := float64(n)
	// ponytail: Suffixes stop at billions; add larger units if counts reach trillions.
	for _, unit := range []struct {
		scale  float64
		suffix string
	}{{1e9, "b"}, {1e6, "m"}, {1e3, "k"}} {
		if math.Abs(value) >= unit.scale {
			return fmt.Sprintf("%.1f%s", value/unit.scale, unit.suffix)
		}
	}
	return fmt.Sprint(n)
}
