# gaze

**See the shape of your code, not the noise.**

Gaze turns PHP source into a compact outline of its contracts: declarations, signatures, and documentation, without implementation bodies. Use it to explore unfamiliar code or give an AI assistant focused context.

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
$ gaze --no-lines CustomerRepository.php
Finds customers by email.
class CustomerRepository
  fn find(string $email): ?Customer
```

PHP is the only supported language today. Go and JavaScript are planned.

## Get started

Requires Go 1.27 or later, CGO, and a C compiler. From a clone of this repository:

```sh
CGO_ENABLED=1 GOFLAGS=-mod=mod go install ./cmd/gaze
```

Run `gaze path/to/file.php` or scan a directory recursively with `gaze path/to/domain`.

## Make it yours

```sh
gaze --all path/to/domain                  # include private and protected members
gaze --full path/to/file.php               # preserve original docblocks
gaze --no-lines --no-docs path/to/domain   # omit line ranges and documentation text
gaze stats                                 # view local scan statistics
```

Directory results are ordered by path, with headings relative to the scanned directory. Files without contracts are omitted, and directory symlinks are not followed. Gaze does not resolve inherited declarations from other files.

### Reading an outline

Compact output uses `fn` as shorthand for `function`, not PHP arrow-function syntax. It merges unambiguous PHPDoc types into signatures, even with `--no-docs`: **displayed types are not necessarily enforced by PHP**. Compact output also omits `@inheritDoc`; use `--full` to retain original docblocks and distinguish declared types.

Imports, attributes, and implementation bodies are omitted. Simple literal defaults are shown; complex expressions become `…`. Syntax errors can yield partial outlines. Output is presentation text, not a stable machine-readable format.

## Local statistics

Each CLI scan records aggregate metrics in `gaze/stats.db` under the user configuration directory (`$XDG_CONFIG_HOME` or `~/.config` on Linux; `~/Library/Application Support` on macOS). **No paths or source content are stored.** `gaze stats` shows today, the last 30 calendar days including today, and all time in local time; it does not record a scan.

Token counts use OpenAI `o200k_base` and are estimates, not API billing. Input and output are counted per file, including directory headings in output. Saved tokens may be negative. Statistics failures are reported on stderr and cause a nonzero exit status. Library calls do not record statistics.

## Use as a Go package

```go
import "github.com/masnathan/gaze"

outline, err := gaze.Outline("service.php", source, gaze.Options{
    IncludeNonPublic: true,
    FullDocs:         true,
})
```

`Outline` returns formatted text; the filename extension selects the language. Filesystem traversal remains the caller's responsibility.

## Develop

```sh
CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./...
CGO_ENABLED=1 GOFLAGS=-mod=mod go build -o gaze ./cmd/gaze
```

Set `CC` if your C compiler is not found automatically. The Makefile uses the existing `vendor` directory. `make dep` replaces it using `go mod vendor`, which can omit C sources and headers required by the Tree-sitter bindings; use module mode as above if vendor-mode builds fail.

## License

[MIT](LICENSE)
