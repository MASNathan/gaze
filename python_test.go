package gaze

import (
	"strings"
	"testing"
)

func TestPythonOutline(t *testing.T) {
	source := []byte(`import os
VALUE: int = secret()
_hidden = 2
class Service(Base):
    """Service docs.
    More detail.
    """
    field: str = build()
    @staticmethod
    def run(x: int = danger()) -> str:
        """Run it."""
        return secret()
    def _private(self): pass
    def __init__(self): pass
async def fetch():
    def nested(): pass
    return 1
`)
	for _, opts := range []Options{{}, {IncludeNonPublic: true, FullDocs: true, NoLines: true}, {NoDocs: true}} {
		got, err := pythonOutlineOptions(source, opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"VALUE: int = …", "class Service(Base):", "field: str = …", "def run(x: int = …) -> str:", "def __init__(self):", "async def fetch():"} {
			if !strings.Contains(got, want) {
				t.Errorf("opts=%+v missing %q in:\n%s", opts, want, got)
			}
		}
		for _, unwanted := range []string{"import os", "secret()", "build()", "danger()", "def nested"} {
			if strings.Contains(got, unwanted) {
				t.Errorf("opts=%+v leaked %q in:\n%s", opts, unwanted, got)
			}
		}
		if strings.Contains(got, "_private") != opts.IncludeNonPublic {
			t.Errorf("visibility: %s", got)
		}
		if strings.Contains(got, "Service docs.") == opts.NoDocs {
			t.Errorf("docs: %s", got)
		}
		if strings.Contains(got, "L4-") == opts.NoLines {
			t.Errorf("lines: %s", got)
		}
	}
}
