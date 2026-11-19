package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

type remoteRepo struct {
	host, owner, repo, ref, subpath string
}

type remoteEntry struct {
	name string
	dir  bool
}

// remoteScan visits supported regular files in a public repository URL.
// ponytail: URL refs occupy one path segment; use a commit SHA for refs containing slashes.
func remoteScan(rawURL string, callback func(name string, source []byte, heading bool)) error {
	r, err := parseRemoteURL(rawURL)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != "https" || req.URL.Hostname() != apiHost(r.host) || req.URL.Port() != "" {
			return fmt.Errorf("remote: unsafe redirect")
		}
		return nil
	}}
	get := func(u string) ([]byte, http.Header, error) {
		resp, err := client.Get(u)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, nil, fmt.Errorf("remote: HTTP %s", resp.Status)
		}
		// ponytail: cap each response at 32 MiB; raise limit if larger files are needed.
		b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
		if err == nil && len(b) > 32<<20 {
			err = fmt.Errorf("remote: response exceeds 32 MiB")
		}
		return b, resp.Header, err
	}
	var visit func(string, string, bool) error
	visit = func(p, name string, heading bool) error {
		entries, data, isDir, err := r.fetch(p, get)
		if err != nil {
			return fmt.Errorf("remote: %s: %w", p, err)
		}
		if !isDir {
			if !supportedFile(name) {
				if !heading {
					return fmt.Errorf("remote: %s: unsupported file type", name)
				}
				return nil
			}
			callback(name, data, heading)
			return nil
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
		for _, entry := range entries {
			if entry.name == "" || entry.name == "." || entry.name == ".." || strings.ContainsAny(entry.name, "/\\") {
				return fmt.Errorf("remote: invalid entry name %q", entry.name)
			}
			if !entry.dir && !supportedFile(entry.name) {
				continue
			}
			childName := entry.name
			if heading {
				childName = name + "/" + entry.name
			}
			if err := visit(path.Join(p, entry.name), childName, true); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(r.subpath, path.Base(r.subpath), false)
}

func apiHost(host string) string {
	switch host {
	case "github.com":
		return "api.github.com"
	case "gitlab.com":
		return "gitlab.com"
	case "codeberg.org":
		return "codeberg.org"
	}
	return ""
}

func parseRemoteURL(raw string) (remoteRepo, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" {
		return remoteRepo{}, fmt.Errorf("remote: invalid repository URL")
	}
	r := remoteRepo{host: strings.ToLower(u.Hostname())}
	if apiHost(r.host) == "" {
		return r, fmt.Errorf("remote: unsupported host")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return r, fmt.Errorf("remote: expected owner and repository")
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || strings.Contains(p, "\\") {
			return r, fmt.Errorf("remote: invalid path")
		}
	}
	r.owner, r.repo = parts[0], strings.TrimSuffix(parts[1], ".git")
	if r.repo == "" {
		return r, fmt.Errorf("remote: invalid repository")
	}
	parts = parts[2:]
	if len(parts) > 0 {
		if r.host == "gitlab.com" && parts[0] == "-" {
			parts = parts[1:]
		}
		if len(parts) == 0 {
			return r, fmt.Errorf("remote: missing repository path")
		}
		valid := parts[0] == "tree" || parts[0] == "blob" || (r.host == "bitbucket.org" && parts[0] == "src") || (r.host == "codeberg.org" && parts[0] == "src")
		if !valid {
			return r, fmt.Errorf("remote: unsupported repository URL path")
		}
		parts = parts[1:]
		if r.host == "codeberg.org" && len(parts) > 0 && (parts[0] == "branch" || parts[0] == "commit" || parts[0] == "tag") {
			parts = parts[1:]
		}
		if len(parts) == 0 {
			return r, fmt.Errorf("remote: missing ref")
		}
		r.ref = parts[0]
		parts = parts[1:]
		r.subpath = strings.Join(parts, "/")
	}
	return r, nil
}

func escapeRemotePath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

type remoteGet func(string) ([]byte, http.Header, error)

func (r remoteRepo) fetch(p string, get remoteGet) ([]remoteEntry, []byte, bool, error) {
	base := "https://" + apiHost(r.host)
	project := url.PathEscape(r.owner + "/" + r.repo)
	var endpoint string
	switch r.host {
	case "github.com":
		endpoint = base + "/repos/" + url.PathEscape(r.owner) + "/" + url.PathEscape(r.repo) + "/contents/" + escapeRemotePath(p)
	case "codeberg.org":
		endpoint = base + "/api/v1/repos/" + url.PathEscape(r.owner) + "/" + url.PathEscape(r.repo) + "/contents/" + escapeRemotePath(p)
	case "gitlab.com":
		endpoint = base + "/api/v4/projects/" + project
		if p != "" {
			fileURL := endpoint + "/repository/files/" + url.PathEscape(p) + "/raw?ref=" + url.QueryEscape(defaultRef(r.ref, "HEAD"))
			data, _, err := get(fileURL)
			if err == nil {
				return nil, data, false, nil
			}
			// A directory returns 404 from the file endpoint; listing below preserves other failures.
			if !strings.Contains(err.Error(), "HTTP 404 ") {
				return nil, nil, false, err
			}
		}
		endpoint += "/repository/tree?per_page=100&path=" + url.QueryEscape(p) + "&ref=" + url.QueryEscape(defaultRef(r.ref, "HEAD"))
	}
	if (r.host == "github.com" || r.host == "codeberg.org") && r.ref != "" {
		endpoint += "?ref=" + url.QueryEscape(r.ref)
	}
	var entries []remoteEntry
	for page := 1; ; page++ {
		if page > 1000 {
			return nil, nil, false, fmt.Errorf("remote: too many pages")
		}
		b, headers, err := get(endpoint)
		if err != nil {
			return nil, nil, false, err
		}
		if r.host == "bitbucket.org" && !strings.Contains(headers.Get("Content-Type"), "json") {
			return nil, b, false, nil
		}
		if r.host == "github.com" || r.host == "codeberg.org" {
			var items []struct {
				Name, Type        string
				Content, Encoding string
			}
			if len(b) > 0 && b[0] == '[' {
				if err := json.Unmarshal(b, &items); err != nil {
					return nil, nil, false, err
				}
			} else {
				var item struct {
					Name, Type        string
					Content, Encoding string
				}
				if err := json.Unmarshal(b, &item); err != nil {
					return nil, nil, false, err
				}
				if item.Type != "file" || item.Encoding != "base64" {
					return nil, nil, false, fmt.Errorf("remote: unsupported file response")
				}
				data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(item.Content, "\n", ""))
				return nil, data, false, err
			}
			if r.host == "github.com" && len(items) >= 1000 {
				return nil, nil, false, fmt.Errorf("remote: GitHub directory may exceed 1000 entries")
			}
			for _, item := range items {
				if item.Type == "file" || item.Type == "dir" {
					entries = append(entries, remoteEntry{item.Name, item.Type == "dir"})
				}
			}
			return entries, nil, true, nil
		}
		if r.host == "gitlab.com" {
			var items []struct{ Name, Type string }
			if err := json.Unmarshal(b, &items); err != nil {
				return nil, nil, false, err
			}
			for _, item := range items {
				if item.Type == "blob" || item.Type == "tree" {
					entries = append(entries, remoteEntry{item.Name, item.Type == "tree"})
				}
			}
			next := headers.Get("X-Next-Page")
			if next == "" {
				return entries, nil, true, nil
			}
			if next != fmt.Sprint(page+1) {
				return nil, nil, false, fmt.Errorf("remote: invalid pagination")
			}
			u, _ := url.Parse(endpoint)
			q := u.Query()
			q.Set("page", next)
			u.RawQuery = q.Encode()
			endpoint = u.String()
		} else {
			var response struct {
				Values []struct{ Path, Type string }
				Next   string
			}
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, nil, false, err
			}
			for _, item := range response.Values {
				if item.Type == "commit_file" || item.Type == "commit_directory" {
					entries = append(entries, remoteEntry{path.Base(item.Path), item.Type == "commit_directory"})
				}
			}
			if response.Next == "" {
				return entries, nil, true, nil
			}
			u, err := url.Parse(response.Next)
			if err != nil || u.Scheme != "https" || u.Host != apiHost(r.host) || u.User != nil || u.Path != strings.Split(endpoint, "?")[0][len(base):] {
				return nil, nil, false, fmt.Errorf("remote: invalid pagination URL")
			}
			endpoint = response.Next
		}
	}
}

func defaultRef(ref, fallback string) string {
	if ref == "" {
		return fallback
	}
	return ref
}
