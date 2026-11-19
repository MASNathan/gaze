package gaze

import (
	"strings"
	"testing"
)

func TestManagedLanguages(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		outline      func([]byte, Options) (string, error)
	}{
		{"java", "/** Docs */\npublic class Example { public int run() { return 42; } private void secret() {} }", javaOutlineOptions},
		{"kotlin", "/** Docs */\nclass Example {\n fun run(): Int { return 42 }\n private fun secret() {}\n}", kotlinOutlineOptions},
		{"csharp", "/** Docs */\npublic class Example { public int run() { return 42; } private void secret() {} }", csharpOutlineOptions},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.outline([]byte(tc.source), Options{NoLines: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Docs", "Example", "run"} {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %q", want, got)
				}
			}
			for _, unwanted := range []string{"return 42", "secret"} {
				if strings.Contains(got, unwanted) {
					t.Errorf("unexpected %q in %q", unwanted, got)
				}
			}
			got, err = tc.outline([]byte(tc.source), Options{NoLines: true, NoDocs: true, IncludeNonPublic: true})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(got, "Docs") || !strings.Contains(got, "secret") {
				t.Errorf("options ignored: %q", got)
			}
		})
	}
}
