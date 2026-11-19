package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masnathan/gaze"
)

func testConfigDir(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
}

func TestRun(t *testing.T) {
	testConfigDir(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"b.php":        "<?php function zebra(): void {}",
		"nested/a.php": "<?php function ant(): void {}",
		"skip.txt":     "not php",
		"empty.php":    "<?php // no contract",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	var out, stderr bytes.Buffer
	if code := run([]string{dir}, &out, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	result := out.String()
	if !strings.HasPrefix(result, "\nb.php:\n") || !strings.Contains(result, "\n\n"+filepath.Join("nested", "a.php")+":\n") {
		t.Fatalf("missing blank lines before filenames: %q", result)
	}
	if !strings.Contains(result, "zebra") || !strings.Contains(result, "ant") || strings.Index(result, "b.php:") > strings.Index(result, "a.php:") || strings.Contains(result, dir) || strings.Contains(result, "empty.php:") || !strings.Contains(result, filepath.Join("nested", "a.php")+":") {
		t.Fatalf("unexpected output: %s", result)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{filepath.Join(dir, "skip.txt")}, &out, &stderr); code == 0 {
		t.Fatal("unsupported file succeeded")
	}
	if code := run(nil, &out, &stderr); code == 0 {
		t.Fatal("missing argument succeeded")
	}
	if code := run([]string{filepath.Join(dir, "missing.php")}, &out, &stderr); code == 0 {
		t.Fatal("missing file succeeded")
	}
	if code := run([]string{filepath.Join(dir, "b.php")}, errorWriter{}, &stderr); code == 0 {
		t.Fatal("write error succeeded")
	}
	file := filepath.Join(dir, "docs.php")
	if err := os.WriteFile(file, []byte("<?php /**\n * @return int\n */ function countItems(): int {}"), 0644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"-vv", file}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "/**") || !strings.Contains(out.String(), "@return int") {
		t.Fatalf("full docs: exit %d, output %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{file}, &out, &stderr); code != 0 || strings.Contains(out.String(), "@return") || !strings.Contains(out.String(), "fn countItems") {
		t.Fatalf("compact docs: exit %d, output %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"stats"}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "Today") || !strings.Contains(out.String(), "Last 30 days") || !strings.Contains(out.String(), "All time") || !strings.Contains(out.String(), "Tokens saved") {
		t.Fatalf("stats: exit %d, output %q, stderr %q", code, out.String(), stderr.String())
	}
	out.Reset()
	if code := run([]string{"--minimal", file}, &out, &stderr); code != 0 || strings.TrimSpace(out.String()) != "fn countItems(): int" {
		t.Fatalf("compact options: exit %d, output %q", code, out.String())
	}
}

func TestRunVerbosity(t *testing.T) {
	testConfigDir(t)
	file := filepath.Join(t.TempDir(), "example.php")
	source := "<?php\n/** Example documentation.\n * More details.\n */\nclass Example {\n public function visible(): void {}\n private function hidden(): void {}\n}\n"
	if err := os.WriteFile(file, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
		opts  gaze.Options
	}{
		{"default", nil, gaze.Options{}},
		{"minimal", []string{"--minimal"}, gaze.Options{NoLines: true, NoDocs: true}},
		{"v", []string{"-v"}, gaze.Options{IncludeNonPublic: true}},
		{"vv", []string{"-vv"}, gaze.Options{FullDocs: true}},
		{"vvv", []string{"-vvv"}, gaze.Options{IncludeNonPublic: true, FullDocs: true}},
		{"combined", []string{"-v", "-vv"}, gaze.Options{IncludeNonPublic: true, FullDocs: true}},
		{"minimal-full", []string{"--minimal", "-vv"}, gaze.Options{NoLines: true, NoDocs: true, FullDocs: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got, stderr bytes.Buffer
			if code := run(append(tc.flags, file), &got, &stderr); code != 0 {
				t.Fatalf("exit %d: %s", code, stderr.String())
			}
			want, err := gaze.Outline(file, []byte(source), tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != want {
				t.Fatalf("got %q, want %q", got.String(), want)
			}
		})
	}
	for _, old := range []string{"--all", "--full", "--no-lines", "--no-docs"} {
		t.Run("reject "+old, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if code := run([]string{old, file}, &out, &stderr); code == 0 {
				t.Fatalf("removed flag %s succeeded", old)
			}
		})
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
