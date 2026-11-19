# Repository guide

- `gaze.Outline` in `gaze.go` is the public library entrypoint; language-specific files extract and format contracts for 22 languages. CLI traversal, remote retrieval and scan accounting live in `cmd/gaze`; library calls do not record stats.
- Compact output merges PHPDoc types into signatures without marking whether PHP enforces them. `-vv` retains docblocks; `--minimal` still uses PHPDoc for compact type inference. Preserve these distinctions when changing extraction.
- Go 1.27, CGO and a C compiler are required. Focused checks: `CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./... -run TestName` or `CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./internal/stats`. Set `CC` to an available compiler if needed. Stats tests isolate configuration via `XDG_CONFIG_HOME`.
- `make test` and `make build` use module mode (`GOFLAGS=-mod=mod`); `make install` first runs `make dep`. `make dep` runs `go mod download` and `go mod vendor`, replacing ignored `vendor/`. Plain vendoring can omit Tree-sitter C sources and headers; use module mode if vendor-mode builds fail.
- CLI scans persist aggregate-only SQLite metrics under `os.UserConfigDir()/gaze/stats.db`; output/token counters include directory headings, and a stats write failure makes the scan fail. `gaze stats` reads without recording a scan. Never store paths or source content in stats.
- Token counts use locally embedded `o200k_base` vocabulary; `internal/tokens.Count` rejects invalid UTF-8. Preserve token-counting semantics and private stats directory/database permissions.
- This is an independent MIT implementation inspired by zat: do not copy zat code, queries, fixtures, snapshots, or documentation.
