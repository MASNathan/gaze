package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type remoteTransport func(*http.Request) (*http.Response, error)

func (f remoteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRemoteScan(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = remoteTransport(func(r *http.Request) (*http.Response, error) {
		body, status := "", http.StatusOK
		switch r.URL.Path {
		case "/repos/team/project/contents/":
			body = `[{"name":"skip.txt","type":"file"},{"name":"lib","type":"dir"},{"name":"a.go","type":"file"}]`
		case "/repos/team/project/contents/lib":
			body = `[{"name":"b.php","type":"file"},{"name":"link.php","type":"symlink"}]`
		case "/repos/team/project/contents/a.go", "/repos/team/project/contents/lib/b.php":
			body = `{"type":"file","encoding":"base64","content":"aGk="}`
		default:
			status, body = http.StatusNotFound, "missing"
		}
		return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	var got []string
	err := remoteScan("https://github.com/team/project", func(name string, source []byte, heading bool) {
		got = append(got, fmt.Sprintf("%s:%s:%t", name, source, heading))
	})
	if err != nil || strings.Join(got, ",") != "a.go:hi:true,lib/b.php:hi:true" {
		t.Fatalf("got %v, %v", got, err)
	}
	got = nil
	err = remoteScan("https://github.com/team/project/blob/main/a.go", func(name string, source []byte, heading bool) {
		got = append(got, fmt.Sprintf("%s:%s:%t", name, source, heading))
	})
	if err != nil || strings.Join(got, ",") != "a.go:hi:false" {
		t.Fatalf("got %v, %v", got, err)
	}
	err = remoteScan("https://github.com/team/project/blob/main/missing.php", func(string, []byte, bool) {})
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("missing HTTP status: %v", err)
	}
}

func TestRemoteProviderResponses(t *testing.T) {
	for _, tc := range []struct {
		r         remoteRepo
		responses []string
	}{
		{remoteRepo{host: "gitlab.com", owner: "team", repo: "project"}, []string{`[{"name":"a.go","type":"blob"}]`, `source`}},
		{remoteRepo{host: "codeberg.org", owner: "team", repo: "project"}, []string{`[{"name":"a.go","type":"file"}]`, `{"type":"file","encoding":"base64","content":"c291cmNl"}`}},
	} {
		t.Run(tc.r.host, func(t *testing.T) {
			i := 0
			get := func(_ string) ([]byte, http.Header, error) {
				if i >= len(tc.responses) {
					t.Fatal("unexpected request")
				}
				body := tc.responses[i]
				i++
				return []byte(body), make(http.Header), nil
			}
			entries, _, dir, err := tc.r.fetch("", get)
			if err != nil || !dir || len(entries) != 1 || entries[0].name != "a.go" {
				t.Fatalf("entries: %v %v %v", entries, dir, err)
			}
			_, data, dir, err := tc.r.fetch("a.go", get)
			if err != nil || dir || string(data) != "source" {
				t.Fatalf("file: %s %v %v", data, dir, err)
			}
		})
	}
}

func TestParseRemoteURL(t *testing.T) {
	for _, raw := range []string{"http://github.com/team/project", "https://user@github.com/team/project", "https://github.com/team/project/tree", "https://github.com/team/../project"} {
		if _, err := parseRemoteURL(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
