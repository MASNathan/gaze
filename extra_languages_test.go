package gaze

import (
	"strings"
	"testing"
)

func TestExtraLanguageOutlines(t *testing.T) {
	tests := []struct {
		name, source         string
		outline              func([]byte, Options) (string, error)
		want, hidden, secret string
	}{
		{"ruby", "class Service\n  def run(x)\n    secret()\n  end\n  private\n  def hidden\n    secret()\n  end\nend\n", rubyOutlineOptions, "def run(x)", "def hidden", "secret()"},
		{"shell", "run() {\n secret\n}\n_hidden() { secret; }\n", shellOutlineOptions, "run()", "_hidden()", "secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, all := range []bool{false, true} {
				got, err := tt.outline([]byte(tt.source), Options{IncludeNonPublic: all, NoLines: true})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(got, tt.want) || strings.Contains(got, tt.hidden) != all || strings.Contains(got, tt.secret) {
					t.Fatalf("all=%v: %s", all, got)
				}
			}
		})
	}
}
