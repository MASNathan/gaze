---
name: gaze-cli
description: Use gaze to explore declarations, signatures, and documentation in local source files, directories, or public GitHub, GitLab.com, and Codeberg URLs before reading implementations.
---

# Gaze CLI

## When to use

Use `gaze` to quickly discover code structure or APIs in supported source files, directories, or public repositories. Read source directly when behavior, imports, exact types, or implementation details matter.

## Workflow

1. Check `gaze` is available; if not, use normal source-reading tools. Do not install software without permission.
2. Run `gaze path/to/file`, `gaze path/to/directory`, or `gaze https://github.com/OWNER/REPOSITORY`. Remote URLs can target a file, directory, or repository root without cloning. Prefer narrow paths to reduce output. Directory scans recurse; output headings identify files.
3. Use `gaze -v path` to include non-public declarations, `gaze -vv path` for original documentation, or `gaze -vvv path` for both. `gaze --minimal path` omits line ranges and documentation text.
4. Inspect relevant source files before drawing conclusions about implementation or making changes. Use `gaze stats` to view local aggregate scan statistics.

## Remote URLs

Pass a public repository URL directly to the CLI; no browser or local clone is needed to generate an outline. For example, explore Gaze itself on GitHub:

```sh
gaze https://github.com/masnathan/gaze
```

Supported URL shapes (replace placeholders with real repository paths):

```sh
# GitHub file or directory
gaze https://github.com/OWNER/REPOSITORY/blob/main/src/lib.rs
gaze https://github.com/OWNER/REPOSITORY/tree/main/src

# GitLab.com directory
gaze https://gitlab.com/OWNER/REPOSITORY/-/tree/main/src

# Codeberg directory
gaze https://codeberg.org/OWNER/REPOSITORY/src/branch/main/src
```

The same options work for local paths and remote URLs. Use `--minimal` for compact context, `-v` for non-public declarations, and `-vv` for original documentation. An empty outline may mean no declarations are visible by default; try `-v`, especially for Rust entry points.

## Limits

- Output omits implementation bodies and imports; inherited declarations are not resolved. Syntax errors can yield partial outlines. Output is not a stable machine-readable format.
- Compact documentation can change wording. In PHP, compact signatures may include PHPDoc types not enforced by PHP, even with `--minimal`; use `-vv` and inspect source to distinguish them.
- Remote scans accept only public HTTPS URLs on GitHub, GitLab.com, and Codeberg. Credentials, explicit ports, query strings, fragments, self-hosted instances, direct raw URLs, GitLab nested groups, and refs containing `/` are unsupported. Use a commit SHA instead of a ref containing `/`. Network requests disclose the requested URL to the provider; never put secrets in URLs.
- Provider rate limits and network errors can interrupt remote scans. Responses over 32 MiB and GitHub directories with 1,000 or more entries are rejected. A scan can produce partial output before failing; check its exit status before treating results as complete.
- Scans store aggregate metrics locally, not paths, URLs, or source content. Token counts use `o200k_base` estimates, not API billing. A statistics write error fails the scan; `gaze stats` and library calls do not record scans.
