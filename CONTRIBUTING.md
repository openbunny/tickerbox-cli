# Contributing

How to build, test, and submit a change to `tickerbox-cli`.

## Contents

- [Development](#development)
- [Commits and pull requests](#commits-and-pull-requests)
- [Developer Certificate of Origin](#developer-certificate-of-origin)
- [Security-sensitive changes](#security-sensitive-changes)
- [Reporting bugs](#reporting-bugs)

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

## Developer Certificate of Origin

Every commit must carry a `Signed-off-by` trailer, certifying you wrote it or
otherwise have the right to submit it under the
[Developer Certificate of Origin](https://developercertificate.org/). Add it
with `git commit --signoff` (or `-s`). This project does not use a Contributor
License Agreement; the DCO is the only requirement.

## Security-sensitive changes

Any `--verbose` or `--debug` flag added in the future must redact password
fields (Wi-Fi and access-point credentials) before they reach a log or the
console; see [SECURITY.md](SECURITY.md) for the reporting process if you find
a case that does not.

## Reporting bugs

Open an issue with the exact command run, the device behaviour observed, and the
expected result. For a security report, follow [SECURITY.md](SECURITY.md)
instead.
