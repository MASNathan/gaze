package gaze

import (
	"strings"
	"testing"
)

func TestCFamilyOutline(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outline func([]byte, Options) (string, error)
		source  string
		want    []string
		omit    []string
	}{
		{"C", cOutlineOptions, "/** API */\nint add(int a, int b) { return a + b; }\nint count = 42;\nstruct Item { int id; };", []string{"API", "int add(int a, int b)", "int count = …;", "struct Item", "int id;"}, []string{"return a + b", "42"}},
		{"C++", cppOutlineOptions, "class Thing { private: int secret; public: int get() { return secret; } };", []string{"class Thing", "int get()"}, []string{"int secret;", "return secret"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.outline([]byte(tc.source), Options{NoLines: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in %q", s, got)
				}
			}
			for _, s := range tc.omit {
				if strings.Contains(got, s) {
					t.Errorf("unexpected %q in %q", s, got)
				}
			}
		})
	}
}

func TestCPPOutlineNonPublic(t *testing.T) {
	got, err := cppOutlineOptions([]byte("class T { int hidden; public: int shown; };"), Options{NoLines: true, IncludeNonPublic: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "int hidden;") || !strings.Contains(got, "int shown;") {
		t.Fatal(got)
	}
}
