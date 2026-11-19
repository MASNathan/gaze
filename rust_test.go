package gaze

import (
	"strings"
	"testing"
)

func TestRustOutline(t *testing.T) {
	source := []byte("/// API\npub struct User { pub name: String, secret: u32 }\nimpl User { pub fn name(&self) -> &str { &self.name } fn hidden(&self) {} }\npub trait Named { fn name(&self) -> &str; }\nfn internal() {}\npub const ANSWER: i32 = 42;\n")
	got, err := rustOutlineOptions(source, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"API", "pub struct User", "pub name: String", "impl User", "pub fn name(&self) -> &str", "pub trait Named", "fn name(&self) -> &str;", "pub const ANSWER: i32 = …"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
	for _, unwanted := range []string{"secret", "hidden", "internal", "42", "&self.name"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("contains %q: %s", unwanted, got)
		}
	}
}
