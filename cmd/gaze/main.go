package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/url"
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
			if err := printStats(stdout, item.name, m); err != nil {
				fmt.Fprintf(stderr, "gaze: stats: %v\n", err)
				return 1
			}
		}
		return 0
	}
	flags := flag.NewFlagSet("gaze", flag.ContinueOnError)
	flags.SetOutput(stderr)
	minimal := flags.Bool("minimal", false, "omit line ranges and documentation text")
	verbose := flags.Bool("v", false, "include non-public members")
	moreVerbose := flags.Bool("vv", false, "preserve full documentation comments")
	mostVerbose := flags.Bool("vvv", false, "include non-public members and preserve full documentation comments")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: gaze [options] <file-or-directory-or-url>\n       gaze stats")
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
	printFile := func(name, headingName string, source []byte, heading bool, err error) {
		m.Files++
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
			outline, err = gaze.Outline(name, source, gaze.Options{IncludeNonPublic: *verbose || *mostVerbose, FullDocs: *moreVerbose || *mostVerbose, NoLines: *minimal, NoDocs: *minimal})
			if err == nil && outline != "" {
				output := outline
				if heading {
					output = "\n" + headingName + ":\n" + outline
				}
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
		if err != nil {
			m.FailedFiles++
			fmt.Fprintf(stderr, "gaze: %s: %v\n", name, err)
			failed = true
		}
	}
	parsed, parseErr := url.Parse(path)
	if parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		err := remoteScan(path, func(name string, source []byte, heading bool) {
			printFile(name, name, source, heading, nil)
		})
		if err != nil {
			fmt.Fprintf(stderr, "gaze: %v\n", err)
			failed = true
		}
	} else if info, err := os.Stat(path); err != nil {
		fmt.Fprintf(stderr, "gaze: %s: %v\n", path, err)
		failed = true
	} else if !info.IsDir() && !supportedFile(path) {
		fmt.Fprintf(stderr, "gaze: %s: unsupported file type\n", path)
		failed = true
	} else if !info.IsDir() {
		source, err := os.ReadFile(path)
		printFile(path, path, source, false, err)
	} else {
		err := filepath.WalkDir(path, func(name string, entry fs.DirEntry, walkErr error) error {
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
				source, readErr := os.ReadFile(name)
				relative, relErr := filepath.Rel(path, name)
				if readErr == nil && relErr != nil {
					readErr = relErr
				}
				if relErr != nil {
					relative = name
				}
				printFile(name, relative, source, true, readErr)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "gaze: %s: %v\n", path, err)
			failed = true
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
