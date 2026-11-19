package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/masnathan/gaze/internal/stats"
)

func TestPrintStats(t *testing.T) {
	for _, tc := range []struct {
		name    string
		metrics stats.Metrics
		want    string
	}{
		{"empty", stats.Metrics{}, "░░░░░░░░░░░░░░░░░░░░  —"},
		{"saved", stats.Metrics{InputTokens: 100, OutputTokens: 25}, "███████████████░░░░░  75.0%"},
		{"increased", stats.Metrics{InputTokens: 100, OutputTokens: 150}, "░░░░░░░░░░░░░░░░░░░░  -50.0%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := printStats(&out, "Today", tc.metrics); err != nil || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output %q, error %v; want %q", out.String(), err, tc.want)
			}
		})
	}
	var out bytes.Buffer
	if err := printStats(&out, "Today", stats.Metrics{InputTokens: 24000, OutputTokens: 4272}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"82.2%", "24.0k → 4.3k  (19.7k saved)"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output %q; want %q", out.String(), want)
		}
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("non-terminal output contains ANSI escapes: %q", out.String())
	}
	if strings.Contains(out.String(), "Bytes") || strings.Contains(out.String(), "o200k") {
		t.Fatalf("unexpected detail in output: %q", out.String())
	}
	if err := printStats(failingStatsWriter{}, "Today", stats.Metrics{}); err == nil {
		t.Fatal("expected write error")
	}
}

type failingStatsWriter struct{}

func (failingStatsWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
