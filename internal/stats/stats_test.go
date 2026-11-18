package stats

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRecordRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	now := time.Date(2026, 3, 1, 0, 30, 0, 0, time.FixedZone("local", -8*3600))
	today, last, all, err := Read(now)
	if err != nil || today != (Metrics{}) || last != (Metrics{}) || all != (Metrics{}) {
		t.Fatalf("empty read: %v %v %v %v", today, last, all, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "gaze")); !os.IsNotExist(err) {
		t.Fatalf("empty read created directory: %v", err)
	}
	m := Metrics{1, 1, 2, 3, 4, 5, 6, 7, 8}
	for _, date := range []time.Time{now, now.AddDate(0, 0, -29), now.AddDate(0, 0, -30), now.AddDate(0, 0, 1)} {
		if err := Record(m, date); err != nil {
			t.Fatal(err)
		}
	}
	today, last, all, err = Read(now)
	twice, four := m, m
	add(&twice, m)
	add(&four, twice)
	add(&four, m)
	if err != nil || today != m || last != twice || all != four {
		t.Fatalf("read: %v %v %v %v", today, last, all, err)
	}
	for path, mode := range map[string]os.FileMode{filepath.Join(dir, "gaze"): 0700, filepath.Join(dir, "gaze", "stats.db"): 0600} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("permissions %s: %v %v", path, info, err)
		}
	}
}

func TestConcurrentRecord(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	now := time.Now()
	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- Record(Metrics{Scans: 1, FailedScans: 1}, now) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	today, _, _, err := Read(now)
	if err != nil || today.Scans != n || today.FailedScans != n {
		t.Fatalf("got %+v: %v", today, err)
	}
}

func TestMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "gaze"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "gaze", "stats.db")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	db, err := open(path, "rw")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE daily (date TEXT PRIMARY KEY, scans INTEGER NOT NULL, files INTEGER NOT NULL, failed_files INTEGER NOT NULL, input_bytes INTEGER NOT NULL, output_bytes INTEGER NOT NULL, input_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL, duration_ns INTEGER NOT NULL); INSERT INTO daily VALUES ('2026-01-01', 1, 2, 3, 4, 5, 6, 7, 8)`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	today, _, _, err := Read(now)
	if err != nil || today.Scans != 1 || today.FailedScans != 0 {
		t.Fatalf("old schema read: %+v %v", today, err)
	}
	if err := Record(Metrics{Scans: 1, FailedScans: 1}, now); err != nil {
		t.Fatal(err)
	}
	today, _, _, err = Read(now)
	if err != nil || today.Scans != 2 || today.FailedScans != 1 {
		t.Fatalf("migrated: %+v %v", today, err)
	}
}
