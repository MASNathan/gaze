package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSkillDetectsOpenCode2(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	if err := os.WriteFile(filepath.Join(dir, "opencode2"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if !agentAvailable(skillAgent{name: "opencode", binary: "opencode"}) {
		t.Fatal("opencode2 not detected")
	}
}
