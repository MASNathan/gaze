package gaze

import (
	"strings"
	"testing"
)

func TestCompactDoc(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Reads the value from a file and an index.", "Reads value from file and index."},
		{"Use in order to utilize a cache.", "Use to use cache."},
		{"This is basically a utility.", "This is utility."},
		{"Preutilize values.", "Preutilize values."},
		{"Please make sure to initiate the process.", "start process."},
		{"Kindly remember to terminate a task.", "end task."},
		{"The handler is responsible for work.", "handler handles work."},
		{"Actually, THE result is a value.", ", result is value."},
		{"Use `the value` in the example.", "Use `the value` in the example."},
		{"See @param the value.", "See @param the value."},
		{"Don't change the value.", "Don't change value."},
		{"Record adds m to the aggregate for now's local calendar date.", "Record adds m to aggregate for now's local calendar date."},
		{"Read today's and yesterday's values from the cache.", "Read today's and yesterday's values from cache."},
		{"Use 'the value' in the example.", "Use 'the value' in the example."},
	} {
		if got := compactDoc(tc.input); got != tc.want {
			t.Errorf("compactDoc(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestCompactDocumentationOptions(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"x.go", "package p\n// Reads the value from a file.\nfunc Read() {}\n"},
		{"x.js", "/** Reads the value from a file. */\nexport function read() {}\n"},
		{"x.py", "def read():\n    \"\"\"Reads the value from a file.\"\"\"\n    pass\n"},
		{"x.php", "<?php\n/** Reads the value from a file. */\nfunction read() {}\n"},
		{"x.ex", "@doc \"Reads the value from a file.\"\ndef read(), do: :ok\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []struct {
				opts Options
				want string
			}{
				{Options{}, "Reads value from file."},
				{Options{FullDocs: true}, "Reads the value from a file."},
			} {
				got, err := Outline(tc.name, []byte(tc.source), mode.opts)
				if err != nil || !strings.Contains(got, mode.want) {
					t.Errorf("Outline(%+v) = %q, %v; want %q", mode.opts, got, err, mode.want)
				}
			}
		})
	}
}
