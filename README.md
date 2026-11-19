# gaze

**Understand the code. Skip the implementation noise.**

[![CI](https://github.com/masnathan/gaze/actions/workflows/ci.yml/badge.svg)](https://github.com/masnathan/gaze/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/masnathan/gaze)](https://github.com/masnathan/gaze/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Gaze is an open-source CLI and Go library that turns source code into compact outlines: declarations, signatures and documentation, without implementation bodies. Get oriented in an unfamiliar codebase, review an API, or give your AI coding assistant focused context instead of entire files.

- **22 languages** — from Go, PHP and TypeScript to Rust, Python and C++.
- **Files, folders and public repositories** — explore local code or pass a GitHub, GitLab.com or Codeberg URL.
- **Control the detail** — public APIs by default, private declarations and full documentation when you need them.
- **Built for humans and agents** — readable terminal output and installable skills for Claude Code, Codex and OpenCode.
- **Measure your savings** — local token statistics, without storing source code, paths or URLs.

[Install](#install) · [Quick start](#quick-start) · [Agent setup](#agent-skill) · [Languages](#supported-languages) · [Contribute](#contributing)

## See it in action

Source:

```php
/** Finds customers by email. */
class CustomerRepository {
    public function find(string $email): ?Customer {
        return $this->database->lookup($email);
    }
    private function connect(): void {}
}
```

```text
$ gaze --minimal CustomerRepository.php
class CustomerRepository
  fn find(string $email): ?Customer
```

## Install

### Download a binary

No Go toolchain or C compiler needed to use a release binary. Open [the latest release](https://github.com/masnathan/gaze/releases/latest) and download the file for your system:

| System | Architecture | Release file |
| --- | --- | --- |
| Linux | x86-64 | `gaze-linux-amd64` |
| Linux | ARM64 | `gaze-linux-arm64` |
| macOS | Apple Silicon | `gaze-darwin-arm64` |

On Linux or macOS, open a terminal in the download folder. Replace the filename below with the one you downloaded:

```sh
mkdir -p "$HOME/.local/bin"
install -m 755 gaze-linux-amd64 "$HOME/.local/bin/gaze"
export PATH="$HOME/.local/bin:$PATH"
gaze --help
```

Add `export PATH="$HOME/.local/bin:$PATH"` to your shell configuration to keep it available in new terminals. Release binaries are built on Ubuntu 24.04 and macOS 15; older systems may need a source build. Release binaries for Intel macOS and Windows are not currently provided.

### Nix

With flakes enabled:


```sh
nix profile install github:masnathan/gaze
```

### Go

Requires **Go 1.27 or later**, **CGO** and a **C compiler** such as GCC or Clang:

```sh
CGO_ENABLED=1 go install github.com/masnathan/gaze/cmd/gaze@latest
```

Ensure `GOBIN` (or `$(go env GOPATH)/bin` when `GOBIN` is unset) is in your `PATH`, then run `gaze --help`. If compilation cannot find your C compiler, set `CC=gcc` or `CC=clang` to an installed compiler.

Prefer building a checkout? See [development](#develop).

## Quick start

From your project folder:

```sh
# See the public API across your project
gaze .

# Keep only declarations and signatures
gaze --minimal .

# Include private and internal declarations
gaze -v .

# Check token savings from your scans
gaze stats
```

Or explore this repository without cloning it:

```sh
gaze https://github.com/masnathan/gaze
```

Need your AI assistant to use Gaze? Install the [agent skill](#agent-skill) next.

## Agent skill

Run `gaze skill` to detect installed Claude Code, Codex, and OpenCode CLIs, select agents, and choose project or user scope. Use arrow keys to navigate, space to select agents, and Enter to confirm each choice. The command shows destination paths before confirmation. Existing skills at selected destinations are replaced.

For noninteractive installation, specify both options, for example:

```sh
gaze skill --agents=claude,codex --scope=user
```

Supported agent names are `claude`, `codex`, and `opencode`; scope is `project` or `user`. Explicit agent names do not require the corresponding CLI to be on `PATH`.

Update all existing skill installations without prompts:

```sh
gaze skill --update
```

This checks the current project's and user's standard skill locations for Claude Code, Codex, and OpenCode, including OpenCode installations invoked as `opencode2`. It replaces existing skill files but does not create missing installations. If none are found, the command exits with an error. `--update` cannot be combined with `--agents` or `--scope`.

## Start exploring

```sh
# Outline one file
gaze path/to/service.php

# Scan a directory recursively
gaze path/to/project

# Include non-public declarations
gaze -v path/to/main.rs

# Omit line ranges and documentation
gaze --minimal path/to/project

# View local scan statistics
gaze stats
```

Directory results are ordered by path, with headings relative to the scanned directory. Files without contracts are omitted; directory symlinks are not followed. Gaze does not resolve inherited declarations from other files.

### Public repositories

Pass a public repository URL to scan a file or directory without cloning it. GitHub, GitLab.com and Codeberg are supported.

```sh
# A file
gaze https://github.com/OWNER/REPOSITORY/blob/main/src/lib.rs

# A directory or repository root
gaze https://github.com/OWNER/REPOSITORY/tree/main/src
gaze https://github.com/OWNER/REPOSITORY

# GitLab and Codeberg directories
gaze https://gitlab.com/OWNER/REPOSITORY/-/tree/main/src
gaze https://codeberg.org/OWNER/REPOSITORY/src/branch/main/src

# Include non-public declarations in a remote file
gaze -v https://github.com/OWNER/REPOSITORY/blob/main/src/main.rs
```

The same outline options apply to local and remote files. An empty outline can mean the file has no declarations visible by default: try `-v`, especially for Rust entry points with private `main` functions. Non-success HTTP responses report their status on stderr and result in a nonzero exit code. A directory scan can produce partial output before a later request fails.

**Current remote limits:**

- Only public repositories over HTTPS; credentials, explicit ports, query strings, fragments and self-hosted instances are not supported.
- Provider API rate limits and network errors can interrupt scans. Large directories may require many requests.
- GitLab nested groups, direct raw URLs and refs containing `/` are not supported. Use a commit SHA instead of a ref containing `/`.
- GitHub directories with 1,000 or more entries and responses over 32 MiB are rejected.
- Remote scans record aggregate scan and processed-file statistics. A download failure fails the scan but is not counted as a failed file.

### Options

| Option | Output |
| --- | --- |
| Default | Public declarations and compact documentation |
| `--minimal` | Omit line ranges and documentation text |
| `-v` | Include non-public declarations |
| `-vv` | Preserve full documentation |
| `-vvv` | Include non-public declarations and preserve full documentation |

Options combine: `-v -vv` equals `-vvv`; `--minimal` omits documentation and lines even when combined with `-vv`.

## Supported languages

| Language | Extensions |
| --- | --- |
| PHP | `.php` |
| Go | `.go` |
| JavaScript | `.js`, `.mjs`, `.cjs` |
| TypeScript | `.ts`, `.mts`, `.cts`, `.tsx` |
| Python | `.py` |
| Rust | `.rs` |
| Java | `.java` |
| Kotlin | `.kt`, `.kts` |
| C# | `.cs` |
| C | `.c`, `.h` |
| C++ | `.cc`, `.cpp`, `.cxx`, `.hh`, `.hpp`, `.hxx` |
| Ruby | `.rb` |
| Shell | `.sh`, `.bash` |
| Dart | `.dart` |
| Scala | `.scala`, `.sc` |
| Elixir | `.ex`, `.exs` |
| Lua | `.lua` |
| R | `.r` |
| Zig | `.zig` |
| Objective-C | `.m`, `.mm` |
| HTML | `.html`, `.htm` |
| CSS | `.css` |

Ambiguous `.h` headers use the C parser. Shell files use the Bash parser; extensionless scripts are not detected.

## Understanding the outline

Imports and implementation bodies are omitted. Go hides unexported names, Rust hides declarations without visibility modifiers, Python hides single-underscore and name-mangled names (but retains special methods), and JavaScript hides `#` private members. Use `-v` or `-vvv` to include non-public declarations. JavaScript outlines include non-exported top-level declarations.

By default, documentation prose is shortened by removing some English articles and filler words and replacing selected expressions; wording and meaning may change. Text containing code-like markup is left unchanged. Use `-vv` or `-vvv` to retain original documentation.

Compact output uses `fn` as shorthand for `function`, not PHP arrow-function syntax. It merges unambiguous PHPDoc types into signatures, even with `--minimal`: **displayed types are not necessarily enforced by PHP**. Compact output also omits `@inheritDoc`; use `-vv` or `-vvv` to retain original docblocks and distinguish declared types.

PHP shows simple literal defaults and replaces complex expressions with `…`; other languages may redact defaults and initializers. Syntax errors can yield partial outlines. Output is presentation text, not a stable machine-readable format.

## Local statistics

In a terminal, statistics appear in styled panels. Redirected output remains plain text without ANSI styling.

`gaze stats` shows today, the last 30 calendar days (including today), and all time, in local time:

```text
Today
  Tokens saved  ████████████████░░░░  82.2%
  Tokens  24.0k → 4.3k  (19.7k saved)
  Scans  12 (0 failed)   Files  48 (0 failed)   Time  2.4s
```

Savings are `(input tokens - output tokens) / input tokens`. Here, output uses about 17.8% of the input tokens, saving 82.2%. Counts are rounded for display (`k` = thousand, `m` = million, `b` = billion). If output exceeds input, savings are negative; without input tokens, the percentage is `—`.

Token counts use OpenAI `o200k_base` and are estimates, not API billing. Input and output are counted per file, including directory headings in output.

Each CLI scan records aggregate metrics in `gaze/stats.db` under the user configuration directory (`$XDG_CONFIG_HOME` or `~/.config` on Linux; `~/Library/Application Support` on macOS). **No paths, URLs or source content are stored.** `gaze stats` and library calls do not record scans. Statistics failures are reported on stderr and cause a nonzero exit status.

## Go package

```go
import "github.com/masnathan/gaze"

outline, err := gaze.Outline("service.php", source, gaze.Options{
    IncludeNonPublic: true,
    FullDocs:         true,
})
```

`Outline` returns formatted text; the filename extension selects the language. Filesystem traversal and remote retrieval remain the caller's responsibility.

## Releases and package distribution

Run the release workflow manually from GitHub Actions to increment the minor version, update the Nix package, and create the tag and GitHub Release. Release builds target Linux and macOS (amd64 and arm64) and Windows (amd64). Pull requests and branch pushes run Linux tests only.

## Contributing

Bug reports, documentation improvements and language-support fixes are welcome.

- [Report a bug](https://github.com/masnathan/gaze/issues): include your OS, Gaze version, command, a small source example, and expected versus actual output. Remove secrets and private code before posting.
- [Suggest an improvement](https://github.com/masnathan/gaze/issues): describe the use case and the output you would find useful.
- Submit a pull request: keep changes focused, add a regression test for behavior changes, and run the tests below. For larger changes, open an issue first to discuss scope.

Gaze is an independent MIT implementation inspired by zat. Contributions must not copy zat code, queries, fixtures, snapshots or documentation.

## Develop

Requires Go 1.27 or later and a C compiler, with CGO enabled.

```sh
git clone https://github.com/masnathan/gaze.git
cd gaze
CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./...
CGO_ENABLED=1 GOFLAGS=-mod=mod go build -o gaze ./cmd/gaze
./gaze --help
```

Install your checkout with `CGO_ENABLED=1 GOFLAGS=-mod=mod go install ./cmd/gaze`, or `nix profile install path:.` with Nix.

Set `CC` if your C compiler is not found automatically. The Makefile uses module mode for builds and tests. `make dep` replaces the `vendor` directory using `go mod vendor`, which can omit C sources and headers required by the Tree-sitter bindings; use module mode as above if vendor-mode builds fail.

## License

[MIT](LICENSE)
