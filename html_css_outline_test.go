package gaze

import (
	"strings"
	"testing"
)

func TestHTMLOutlineStructure(t *testing.T) {
	source := []byte("<main><h1>Secret text</h1><script>secretCode()</script><style>.secret { color: red }</style><img src=\"secret.png\"></main>")
	got, err := htmlOutlineOptions(source, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	want := "<main>\n  <h1>\n  <script>\n  <style>\n  <img>\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCSSOutlineSelectorsAndCustomProperties(t *testing.T) {
	source := []byte(":root { --accent: secret; color: red; }\n@media (min-width: 1px) { .card, #item:hover { --size: 3px; padding: 10px; } }")
	got, err := cssOutlineOptions(source, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{":root", "--accent", ".card, #item:hover", "--size"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	for _, unwanted := range []string{"secret", "color", "red", "3px", "padding"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in %q", unwanted, got)
		}
	}
}
