package gaze

import (
	"strings"
	"testing"
)

func TestJavaScriptOutline(t *testing.T) {
	source := []byte(`/** API summary */
export class Example extends Base {
  #secret = expensive();
  value = expensive();
  /** Run operation */
  async run(x) { return hidden(); }
  #internal() { return hidden(); }
}
export const add = (a, b) => a + b, count = compute();
function helper(arg) { return implementation(); }
`)
	for _, tc := range []struct {
		name   string
		opts   Options
		want   []string
		absent []string
	}{
		{"compact", Options{}, []string{"API summary", "export class Example extends Base", "value = …", "async run(x)", "export const add = (a, b) => …", "function helper(arg)"}, []string{"#secret", "#internal", "expensive", "implementation", "compute()"}},
		{"all", Options{IncludeNonPublic: true, FullDocs: true}, []string{"/** API summary */", "#secret = …", "#internal()"}, []string{"expensive"}},
		{"no docs or lines", Options{NoDocs: true, NoLines: true}, []string{"async run(x)"}, []string{"API summary", "Run operation", "L6"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := javascriptOutlineOptions(source, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %q", want, got)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(got, absent) {
					t.Errorf("unexpected %q in %q", absent, got)
				}
			}
		})
	}
}
