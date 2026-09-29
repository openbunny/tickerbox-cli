# Contributing

## Development

- Use the Go toolchain declared in `go.mod`.
- Build and test: `go build ./...`, `go test -race ./...`.
- Format with `gofumpt`. Lint with `go vet`, `staticcheck`, and
  `golangci-lint run`. CI runs all four; a change lands only when they pass.
- Every change ships tests for the behaviour it adds or fixes. A decoder that
  parses a device payload also ships a fuzz target.
- No command reaches the real device in a test; use `httptest`.

## Commits and pull requests

- One logical change per commit. Use Conventional Commits headers: `feat:`,
  `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.
- The pull request states what changed and why, and links related issues.
- Do not weaken a lint rule, delete a failing test, or widen a suppression to
  make CI pass. Fix the cause, or say why the gate is wrong.

## Reporting bugs

Open an issue with the exact command run, the device behaviour observed, and the
expected result. For a security report, follow [SECURITY.md](SECURITY.md)
instead.
