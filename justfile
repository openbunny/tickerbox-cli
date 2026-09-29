set shell := ["bash", "-uc"]

default:
    just --list

build:
    go build ./...

test:
    go test ./...

test-race:
    go test -race ./...

fuzz fuzztime="10s":
    #!/usr/bin/env bash
    set -euo pipefail
    found=0
    for pkg in $(go list ./...); do
        for fn in $(go test "$pkg" -list '^Fuzz' 2>/dev/null | grep -E '^Fuzz' || true); do
            found=1
            go test "$pkg" -run '^$' -fuzz "^${fn}\$" -fuzztime "{{ fuzztime }}"
        done
    done
    if [ "$found" -eq 0 ]; then
        echo "fuzz: no Fuzz targets found" >&2
        exit 1
    fi

fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    out=$(gofumpt -l .)
    if [ -n "$out" ]; then
        echo "$out"
        echo "fmt-check: run 'gofumpt -w .' to fix" >&2
        exit 1
    fi

vet:
    go vet ./...

staticcheck:
    staticcheck ./...

golangci-lint:
    golangci-lint run

lint: fmt-check staticcheck golangci-lint

coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out

completions:
    mkdir -p dist/completions
    go run . completion bash > dist/completions/tickerbox.bash
    go run . completion zsh > dist/completions/tickerbox.zsh
    go run . completion fish > dist/completions/tickerbox.fish
    go run . completion powershell > dist/completions/tickerbox.ps1

man:
    go run ./docs/gen -man

cli-docs:
    go run ./docs/gen

release-dry-run:
    goreleaser release --snapshot --clean
