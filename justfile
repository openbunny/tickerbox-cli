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

actions-lint:
    actionlint

pinact-check:
    #!/usr/bin/env bash
    set -euo pipefail
    shopt -s nullglob
    files=(.github/workflows/*.yml .github/workflows/*.yaml)
    if [ "${#files[@]}" -eq 0 ]; then
        echo "pinact-check: no workflow files found in .github/workflows" >&2
        exit 1
    fi
    pinact run --check --verify-comment "${files[@]}"

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

demo:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v vhs >/dev/null || { echo "demo: vhs not installed; see https://github.com/charmbracelet/vhs#installation" >&2; exit 1; }
    dir=$(mktemp -d)
    trap 'kill "${mock_pid:-0}" 2>/dev/null || true; rm -rf "$dir"' EXIT
    go build -o "$dir/tickerbox" .
    go run demo/mock.go &
    mock_pid=$!
    retries=50
    interval=0.1
    for _ in $(seq 1 "$retries"); do
        curl -sf http://127.0.0.1:8765/rest/features >/dev/null 2>&1 && break
        sleep "$interval"
    done
    PATH="$dir:$PATH" TICKERBOX_HOST="http://127.0.0.1:8765" vhs demo/tickerbox.tape
    ls -lh demo/tickerbox.gif
