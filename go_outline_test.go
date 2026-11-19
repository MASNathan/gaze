package gaze

import (
	"strings"
	"testing"
)

func TestGoOutlineOptions(t *testing.T) {
	source := []byte(`package example
import "fmt"
// Thing documents Thing.
type Thing[T any] struct {
 Visible, hidden string
 private int
 *Other
 *privateType
}
// Run performs work.
func (x *Thing[T]) Run(v T) error { fmt.Println(v); return nil }
func hidden() {}
const ( Public = 123; private = 42 )
`)
	got, err := goOutlineOptions(source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"L1 package example", "Thing documents Thing.", "type Thing[T any] struct", "Visible", "*Other", "L11 func (x *Thing[T]) Run(v T) error", "const Public"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	for _, absent := range []string{"fmt", "hidden", "private", "123", "Println"} {
		if strings.Contains(got, absent) {
			t.Errorf("unexpected %q in %s", absent, got)
		}
	}
	got, err = goOutlineOptions(source, Options{IncludeNonPublic: true, FullDocs: true, NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package example", "// Thing documents Thing.", "hidden", "private", "func hidden()"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "L10 ") {
		t.Errorf("unexpected line number: %s", got)
	}
	got, err = goOutlineOptions(source, Options{FullDocs: true, NoDocs: true})
	if err != nil || strings.Contains(got, "documents Thing") {
		t.Fatalf("NoDocs: %q, %v", got, err)
	}
}

func TestGoOutlinePackageOnly(t *testing.T) {
	for _, tc := range []struct {
		opts Options
		want string
	}{
		{Options{}, "L1 package p\n"},
		{Options{NoLines: true}, "package p\n"},
	} {
		got, err := goOutlineOptions([]byte("package p\n"), tc.opts)
		if err != nil || got != tc.want {
			t.Fatalf("got %q, %v; want %q", got, err, tc.want)
		}
	}
}

func TestGoOutlineIncomplete(t *testing.T) {
	got, err := goOutlineOptions([]byte("package p\nfunc Public() {"), Options{})
	if err != nil || !strings.Contains(got, "func Public()") {
		t.Fatalf("got %q, %v", got, err)
	}
}
