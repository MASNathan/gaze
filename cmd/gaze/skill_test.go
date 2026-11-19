package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masnathan/gaze"
)

func TestInstallSkill(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := installSkill([]string{"-agents=claude,opencode", "-scope=user"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("install: %s", stderr.String())
	}
	for _, p := range []string{filepath.Join(dir, ".claude/skills/gaze-cli/SKILL.md"), filepath.Join(configDir, "opencode/skills/gaze-cli/SKILL.md")} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents/skills/gaze-cli/SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("unselected agent installed")
	}
	path := filepath.Join(dir, ".claude/skills/gaze-cli/SKILL.md")
	if err := os.WriteFile(path, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := installSkill([]string{"-update"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("update: %s", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, gaze.Skill) {
		t.Fatalf("skill not updated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents/skills/gaze-cli/SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("update installed unselected skill")
	}
}
