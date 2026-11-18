package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	if code := run([]string{"--full", file}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "/**") || !strings.Contains(out.String(), "@return int") {
		t.Fatalf("full docs: exit %d, output %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{file}, &out, &stderr); code != 0 || strings.Contains(out.String(), "@return") || !strings.Contains(out.String(), "fn countItems") {
		t.Fatalf("compact docs: exit %d, output %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"stats"}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "Today") || !strings.Contains(out.String(), "Last 30 days") || !strings.Contains(out.String(), "All time") || !strings.Contains(out.String(), "Tokens (o200k_base estimate)") {
		t.Fatalf("stats: exit %d, output %q, stderr %q", code, out.String(), stderr.String())
	}
	out.Reset()
	if code := run([]string{"--no-lines", "--no-docs", file}, &out, &stderr); code != 0 || strings.TrimSpace(out.String()) != "fn countItems(): int" {
		t.Fatalf("compact options: exit %d, output %q", code, out.String())
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
