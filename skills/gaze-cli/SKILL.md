---
name: gaze-cli
description: Use gaze to explore source-code declarations, signatures, and documentation before reading implementations.
---

# Gaze CLI

## When to use

Use `gaze` to quickly discover code structure or APIs in supported source files, directories, or public repositories. Read source directly when behavior, imports, exact types, or implementation details matter.

## Workflow

1. Check `gaze` is available; if not, use normal source-reading tools. Do not install software without permission.
2. Run `gaze path/to/file` or `gaze path/to/directory`. Prefer narrow paths to reduce output. Directory scans recurse; output headings identify files.
3. Use `gaze -v path` to include non-public declarations, `gaze -vv path` for original documentation, or `gaze -vvv path` for both. `gaze --minimal path` omits line ranges and documentation text.
4. Inspect relevant source files before drawing conclusions about implementation or making changes. Use `gaze stats` to view local aggregate scan statistics.

## Limits

- Output omits implementation bodies and imports; inherited declarations are not resolved. Syntax errors can yield partial outlines. Output is not a stable machine-readable format.
- Compact documentation can change wording. In PHP, compact signatures may include PHPDoc types not enforced by PHP, even with `--minimal`; use `-vv` and inspect source to distinguish them.
- Remote scans accept public HTTPS GitHub, GitLab.com, and Codeberg repository URLs, subject to provider and gaze limits. Network requests disclose the requested URL to the provider.
- Scans store aggregate metrics locally. A statistics write error fails the scan.
