package gaze

import (
	"strings"
	"testing"
)

func TestDartOutline(t *testing.T) {
	source := []byte("/// Docs\nclass Example { String run(int count) { return 'ok'; } void _secret() {} }\nint top() => 1;")
	got, err := dartOutlineOptions(source, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Docs", "class Example", "String run(int count)", "int top()"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	for _, unwanted := range []string{"return 'ok'", "_secret", "=> 1"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in %q", unwanted, got)
		}
	}
	got, err = dartOutlineOptions(source, Options{NoLines: true, NoDocs: true, IncludeNonPublic: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Docs") || !strings.Contains(got, "_secret") {
		t.Errorf("options ignored: %q", got)
	}
}
