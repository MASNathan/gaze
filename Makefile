.PHONY: install uninstall build test dep dep-update

install: dep
	@if [ -n "$$CC" ] || command -v gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOFLAGS=-mod=mod go install ./cmd/gaze; \
	elif command -v nix-shell >/dev/null 2>&1; then \
		nix-shell -p gcc --run 'CGO_ENABLED=1 GOFLAGS=-mod=mod go install ./cmd/gaze'; \
	else \
		echo 'C compiler required: install gcc or set CC' >&2; exit 1; \
	fi

uninstall:
	@bin="$$(go env GOBIN)" || exit 1; \
	if [ -z "$$bin" ]; then \
		gopath="$$(go env GOPATH)" || exit 1; \
		[ -n "$$gopath" ] || exit 1; \
		bin="$${gopath%%:*}/bin"; \
	fi; \
	ext="$$(go env GOEXE)" || exit 1; \
	rm -f -- "$$bin/gaze$$ext"

build:
	@if [ -n "$$CC" ] || command -v gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOFLAGS=-mod=mod go build -o gaze ./cmd/gaze; \
	elif command -v nix-shell >/dev/null 2>&1; then \
		nix-shell -p gcc --run 'CGO_ENABLED=1 GOFLAGS=-mod=mod go build -o gaze ./cmd/gaze'; \
	else \
		echo 'C compiler required: install gcc or set CC' >&2; exit 1; \
	fi

test:
	@if [ -n "$$CC" ] || command -v gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./...; \
	elif command -v nix-shell >/dev/null 2>&1; then \
		nix-shell -p gcc --run 'CGO_ENABLED=1 GOFLAGS=-mod=mod go test ./...'; \
	else \
		echo 'C compiler required: install gcc or set CC' >&2; exit 1; \
	fi

dep:
	go mod download
	go mod vendor

dep-update:
	go get -u ./...
	go mod tidy
	$(MAKE) dep
