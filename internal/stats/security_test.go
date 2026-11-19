package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExistingDirectoryNotChanged(t *testing.T) {
	dir := testConfigDir(t)
	path := filepath.Join(dir, "gaze")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Record(Metrics{}, time.Now()); err == nil {
		t.Fatal("expected error for non-private directory")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("directory changed: %v %v", info, err)
	}
}

func TestSymlinksRejected(t *testing.T) {
	dir := testConfigDir(t)
	target := t.TempDir()
	path := filepath.Join(dir, "gaze")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := Record(Metrics{}, time.Now()); err == nil {
		t.Fatal("expected directory symlink rejection")
	}
	if _, _, _, err := Read(time.Now()); err == nil {
		t.Fatal("expected directory symlink rejection on read")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(target, "db"), filepath.Join(path, "stats.db")); err != nil {
		t.Fatal(err)
	}
	if err := Record(Metrics{}, time.Now()); err == nil {
		t.Fatal("expected database symlink rejection")
	}
	if _, _, _, err := Read(time.Now()); err == nil {
		t.Fatal("expected database symlink rejection on read")
	}
}
