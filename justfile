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
    command -v docker >/dev/null || { echo "demo: docker not installed; see https://docs.docker.com/get-docker/" >&2; exit 1; }
    port=8765
    lsof -ti ":$port" | xargs -r kill -9 2>/dev/null || true
    dir=$(mktemp -d)
    # go run execs the compiled mock as a child, so killing its own pid leaves
    # that child bound to the port; killing by port reaches the real listener
    trap 'lsof -ti ":$port" | xargs -r kill -9 2>/dev/null || true; rm -rf "$dir"' EXIT
    case "$(uname -m)" in
        arm64 | aarch64) goarch=arm64 ;;
        x86_64 | amd64) goarch=amd64 ;;
        *)
            echo "demo: unsupported host arch $(uname -m)" >&2
            exit 1
            ;;
    esac
    # vhs runs inside the linux container below (not the host's Chrome), so the
    # binary it shells out to must be built for linux on the container's arch,
    # which docker run below defaults to matching the host's
    GOOS=linux GOARCH="$goarch" go build -o "$dir/tickerbox" .
    go run demo/mock.go &
    retries=50
    interval=0.1
    for _ in $(seq 1 "$retries"); do
        curl -sf "http://127.0.0.1:$port/rest/features" >/dev/null 2>&1 && break
        sleep "$interval"
    done
    docker run --rm -v "$PWD":/vhs -v "$dir/tickerbox":/usr/local/bin/tickerbox:ro \
        -e "TICKERBOX_HOST=http://tickerbox.local:$port" --add-host tickerbox.local:host-gateway \
        ghcr.io/charmbracelet/vhs demo/tickerbox.tape
    ls -lh demo/tickerbox.gif
