package gaze

import (
	"strings"
	"testing"
)

func TestOutline(t *testing.T) {
	source := []byte("<?php class Example { private function hidden() {} public function show(): void {} }")
	for _, tc := range []struct {
		all         bool
		wantPrivate bool
	}{{false, false}, {true, true}} {
		got, err := Outline("example.PHP", source, Options{IncludeNonPublic: tc.all})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "show") || strings.Contains(got, "hidden") != tc.wantPrivate {
			t.Fatalf("unexpected outline: %s", got)
		}
	}
	documented := []byte("<?php /**\n * @return int\n */ function countItems(): int {}")
	compact, err := Outline("test.php", documented, Options{})
	if err != nil {
		t.Fatal(err)
	}
	full, err := Outline("test.php", documented, Options{FullDocs: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compact, "\n * @return") || !strings.Contains(full, "\n * @return") {
		t.Fatalf("unexpected docs: compact=%q full=%q", compact, full)
	}
	if _, err := Outline("example.js", source, Options{}); err == nil {
		t.Fatal("unsupported language succeeded")
	}
}

func TestOutlineOptions(t *testing.T) {
	source := []byte("<?php\n/**\n * Summary\n * @param string $x\n * @return int\n */\nfunction example($x) {}\n")
	for _, tc := range []struct {
		name         string
		opts         Options
		want, absent string
	}{
		{"default", Options{}, "L7 fn example(string $x): int", "function example"},
		{"no docs", Options{NoDocs: true}, "L7 fn example(string $x): int", "Summary"},
		{"no lines", Options{NoLines: true}, "fn example(string $x): int", "L7"},
		{"combined", Options{NoDocs: true, NoLines: true}, "fn example(string $x): int", "Summary"},
		{"full no docs", Options{FullDocs: true, NoDocs: true, NoLines: true}, "function example($x)", "Summary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Outline("example.php", source, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tc.want) || strings.Contains(got, tc.absent) {
				t.Fatalf("unexpected outline: %q", got)
			}
			if tc.opts.NoLines && strings.Contains(got, "L7") {
				t.Fatalf("line leaked: %q", got)
			}
		})
	}
}
