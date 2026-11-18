// Package stats stores aggregate scan metrics by local calendar date.
package stats

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Metrics contains aggregate scan measurements.
type Metrics struct {
	Scans, FailedScans, Files, FailedFiles, InputBytes, OutputBytes, InputTokens, OutputTokens, DurationNS int64
}

// Record adds m to the aggregate for now's local calendar date.
func Record(m Metrics, now time.Time) error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create stats directory: %w", err)
	}
	info, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("stat stats directory: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("stats directory must be private and not a symlink")
	}
	info, err = os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat stats database: %w", err)
	}
	if err == nil && (!info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0) {
		return fmt.Errorf("stats database must be private and not a symlink")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("create stats database: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close stats database file: %w", err)
	}
	db, err := open(path, "rw")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS daily (
 date TEXT PRIMARY KEY, scans INTEGER NOT NULL, files INTEGER NOT NULL, failed_files INTEGER NOT NULL,
 input_bytes INTEGER NOT NULL, output_bytes INTEGER NOT NULL, input_tokens INTEGER NOT NULL,
 output_tokens INTEGER NOT NULL, duration_ns INTEGER NOT NULL)`)
	if err != nil {
		return fmt.Errorf("create stats table: %w", err)
	}
	// Concurrent writers may both attempt this migration; only one can add the column.
	if _, err = db.Exec(`ALTER TABLE daily ADD COLUMN failed_scans INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name: failed_scans") {
		return fmt.Errorf("migrate stats table: %w", err)
	}
	_, err = db.Exec(`INSERT INTO daily (date, scans, failed_scans, files, failed_files, input_bytes, output_bytes, input_tokens, output_tokens, duration_ns) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT(date) DO UPDATE SET scans=scans+excluded.scans, failed_scans=failed_scans+excluded.failed_scans, files=files+excluded.files,
 failed_files=failed_files+excluded.failed_files, input_bytes=input_bytes+excluded.input_bytes,
 output_bytes=output_bytes+excluded.output_bytes, input_tokens=input_tokens+excluded.input_tokens,
 output_tokens=output_tokens+excluded.output_tokens, duration_ns=duration_ns+excluded.duration_ns`,
		now.Format("2006-01-02"), m.Scans, m.FailedScans, m.Files, m.FailedFiles, m.InputBytes, m.OutputBytes, m.InputTokens, m.OutputTokens, m.DurationNS)
	if err != nil {
		return fmt.Errorf("record stats: %w", err)
	}
	return nil
}

// Read returns today's, the last 30 calendar days' (including today), and all-time totals.
// Dates use now's location; future dates are included only in all-time totals.
func Read(now time.Time) (today, last30, all Metrics, err error) {
	path, err := dbPath()
	if err != nil {
		return today, last30, all, err
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if os.IsNotExist(err) {
		return today, last30, all, nil
	}
	if err != nil {
		return today, last30, all, fmt.Errorf("stat stats directory: %w", err)
	}
	if !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		return today, last30, all, fmt.Errorf("stats directory must be private and not a symlink")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return today, last30, all, nil
	}
	if err != nil {
		return today, last30, all, fmt.Errorf("stat stats database: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return today, last30, all, fmt.Errorf("stats database must be private and not a symlink")
	}
	db, err := open(path, "ro")
	if err != nil {
		return today, last30, all, err
	}
	defer db.Close()
	var hasFailedScans int
	if err := db.QueryRow(`SELECT count(*) FROM pragma_table_info('daily') WHERE name = 'failed_scans'`).Scan(&hasFailedScans); err != nil {
		return today, last30, all, fmt.Errorf("inspect stats table: %w", err)
	}
	failedScans := "0"
	if hasFailedScans != 0 {
		failedScans = "failed_scans"
	}
	rows, err := db.Query(`SELECT date, scans, ` + failedScans + `, files, failed_files, input_bytes, output_bytes, input_tokens, output_tokens, duration_ns FROM daily`)
	if err != nil {
		return today, last30, all, fmt.Errorf("read stats: %w", err)
	}
	defer rows.Close()
	end, start := now.Format("2006-01-02"), now.AddDate(0, 0, -29).Format("2006-01-02")
	for rows.Next() {
		var date string
		var m Metrics
		if err := rows.Scan(&date, &m.Scans, &m.FailedScans, &m.Files, &m.FailedFiles, &m.InputBytes, &m.OutputBytes, &m.InputTokens, &m.OutputTokens, &m.DurationNS); err != nil {
			return Metrics{}, Metrics{}, Metrics{}, fmt.Errorf("scan stats: %w", err)
		}
		add(&all, m)
		if date >= start && date <= end {
			add(&last30, m)
		}
		if date == end {
			add(&today, m)
		}
	}
	if err := rows.Err(); err != nil {
		return Metrics{}, Metrics{}, Metrics{}, fmt.Errorf("iterate stats: %w", err)
	}
	return today, last30, all, nil
}

func add(dst *Metrics, m Metrics) {
	dst.Scans += m.Scans
	dst.FailedScans += m.FailedScans
	dst.Files += m.Files
	dst.FailedFiles += m.FailedFiles
	dst.InputBytes += m.InputBytes
	dst.OutputBytes += m.OutputBytes
	dst.InputTokens += m.InputTokens
	dst.OutputTokens += m.OutputTokens
	dst.DurationNS += m.DurationNS
}

func dbPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(dir, "gaze", "stats.db"), nil
}

func open(path, mode string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	q := u.Query()
	q.Set("mode", mode)
	q.Set("_busy_timeout", "10000")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, fmt.Errorf("open stats database: %w", err)
	}
	return db, nil
}
