package gaze

import (
	"strings"
	"testing"
)

func TestLuaAndROutlines(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		outline            func([]byte, Options) (string, error)
	}{
		{"lua", "-- docs\nfunction hello(x)\n  return secret(x)\nend\nlocal function hidden() end\n", "function hello(x)", luaOutlineOptions},
		{"r", "# docs\nhello <- function(x = secret()) {\n secret(x)\n}\n.hidden <- function() 1\n", "hello <- function(x = …)", rOutlineOptions},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.outline([]byte(tc.source), Options{NoLines: true})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tc.want) || strings.Contains(got, "secret") || strings.Contains(got, "hidden") || !strings.Contains(got, "docs") {
				t.Fatalf("unexpected outline: %s", got)
			}
			got, err = tc.outline([]byte(tc.source), Options{NoLines: true, NoDocs: true, IncludeNonPublic: true})
			if err != nil || strings.Contains(got, "docs") || !strings.Contains(got, "hidden") {
				t.Fatalf("options: %s, %v", got, err)
			}
		})
	}
}
