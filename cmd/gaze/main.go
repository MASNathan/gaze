package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/masnathan/gaze"
	"github.com/masnathan/gaze/internal/stats"
	"github.com/masnathan/gaze/internal/tokens"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func supportedFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".php", ".go", ".js", ".mjs", ".cjs", ".py",
		".ts", ".tsx", ".mts", ".cts", ".rs", ".java", ".kt", ".kts", ".cs",
		".c", ".h", ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx",
		".rb", ".sh", ".bash", ".dart", ".scala", ".sc", ".ex", ".exs",
		".lua", ".r", ".zig", ".m", ".mm", ".html", ".htm", ".css":
		return true
	}
	return false
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "stats" {
		today, month, all, err := stats.Read(time.Now())
		if err != nil {
			fmt.Fprintf(stderr, "gaze: stats: %v\n", err)
			return 1
		}
		for _, item := range []struct {
			name string
			m    stats.Metrics
		}{{"Today", today}, {"Last 30 days", month}, {"All time", all}} {
			m := item.m
			if _, err := fmt.Fprintf(stdout, "%s\n  Scans: %d (failed: %d)\n  Files: %d (failed: %d)\n  Bytes: %d input, %d output\n  Tokens (o200k_base estimate): %d input, %d output, %d saved\n  Processing time: %s\n", item.name, m.Scans, m.FailedScans, m.Files, m.FailedFiles, m.InputBytes, m.OutputBytes, m.InputTokens, m.OutputTokens, m.InputTokens-m.OutputTokens, time.Duration(m.DurationNS)); err != nil {
				fmt.Fprintf(stderr, "gaze: stats: %v\n", err)
				return 1
			}
		}
		return 0
	}
	flags := flag.NewFlagSet("gaze", flag.ContinueOnError)
	flags.SetOutput(stderr)
	all := flags.Bool("all", false, "include private and protected members")
	full := flags.Bool("full", false, "preserve full documentation comments")
	noLines := flags.Bool("no-lines", false, "omit line ranges")
	noDocs := flags.Bool("no-docs", false, "omit documentation text")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: gaze [--all] [--full] [--no-lines] [--no-docs] <file-or-directory>\n       gaze stats")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 1
	}
	started := time.Now()
	m := stats.Metrics{Scans: 1}
	failed := false
	path := flags.Arg(0)
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "gaze: %s: %v\n", path, err)
		failed = true
	} else if !info.IsDir() && !supportedFile(path) {
		fmt.Fprintf(stderr, "gaze: %s: unsupported file type\n", path)
		failed = true
	} else {
		printFile := func(name string, heading bool) {
			m.Files++
			source, err := os.ReadFile(name)
			if err == nil {
				m.InputBytes += int64(len(source))
				var count int
				count, err = tokens.Count(source)
				if err == nil {
					m.InputTokens += int64(count)
				}
			}
			if err == nil {
				var outline string
				outline, err = gaze.Outline(name, source, gaze.Options{IncludeNonPublic: *all, FullDocs: *full, NoLines: *noLines, NoDocs: *noDocs})
				if err == nil && outline != "" {
					output := outline
					if heading {
						var relative string
						relative, err = filepath.Rel(path, name)
						if err == nil {
							output = "\n" + relative + ":\n" + outline
						}
					}
					if err == nil {
						var n int
						n, err = io.WriteString(stdout, output)
						m.OutputBytes += int64(n)
						if n > 0 {
							count, countErr := tokens.Count([]byte(output[:n]))
							if countErr != nil {
								err = countErr
							} else {
								m.OutputTokens += int64(count)
							}
						}
					}
				}
			}
			if err != nil {
				m.FailedFiles++
				fmt.Fprintf(stderr, "gaze: %s: %v\n", name, err)
				failed = true
			}
		}
		if !info.IsDir() {
			printFile(path, false)
		} else {
			err = filepath.WalkDir(path, func(name string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					fmt.Fprintf(stderr, "gaze: %s: %v\n", name, walkErr)
					failed = true
					if entry != nil && !entry.IsDir() {
						m.Files++
						m.FailedFiles++
					}
					return nil
				}
				if entry.Type().IsRegular() && supportedFile(name) {
					printFile(name, true)
				}
				return nil
			})
			if err != nil {
				fmt.Fprintf(stderr, "gaze: %s: %v\n", path, err)
				failed = true
			}
		}
	}
	if failed {
		m.FailedScans = 1
	}
	m.DurationNS = time.Since(started).Nanoseconds()
	if err := stats.Record(m, started); err != nil {
		fmt.Fprintf(stderr, "gaze: stats: %v\n", err)
		failed = true
	}
	if failed {
		return 1
	}
	return 0
}
