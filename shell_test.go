package gaze

import (
	"strings"
	"testing"
)

func TestShellDocumentationAndVariables(t *testing.T) {
	source := []byte("#!/bin/sh\n# API token\n# Keep private\nexport TOKEN=$(secret) _PRIVATE=hidden\nUNRELATED=secret\n# disconnected\n\nIGNORED=secret\n# Run command\nrun() { secret; }\n")
	for _, opts := range []Options{{}, {NoDocs: true, NoLines: true}, {FullDocs: true, IncludeNonPublic: true}} {
		got, err := shellOutlineOptions(source, opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"TOKEN=…", "run()"} {
			if !strings.Contains(got, want) {
				t.Errorf("opts=%+v missing %q: %s", opts, want, got)
			}
		}
		for _, unwanted := range []string{"secret", "UNRELATED", "IGNORED", "disconnected", "#!/bin/sh"} {
			if strings.Contains(got, unwanted) {
				t.Errorf("opts=%+v leaked %q: %s", opts, unwanted, got)
			}
		}
		if strings.Contains(got, "_PRIVATE=…") != opts.IncludeNonPublic {
			t.Errorf("visibility: %s", got)
		}
		if strings.Contains(got, "API token") == opts.NoDocs {
			t.Errorf("docs: %s", got)
		}
		if strings.Contains(got, "# API token") != opts.FullDocs {
			t.Errorf("full docs: %s", got)
		}
		if strings.Contains(got, "L4 ") == opts.NoLines {
			t.Errorf("lines: %s", got)
		}
	}
}
