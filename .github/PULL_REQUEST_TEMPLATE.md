## What

What this change does.

## Why

The problem it solves or the reason for it.

## Checklist

- [ ] `go build ./...`, `go test -race ./...` pass
- [ ] `gofumpt`, `go vet`, `staticcheck`, `golangci-lint run` clean
- [ ] Tests cover the added or fixed behaviour
- [ ] Every commit carries a `Signed-off-by` trailer (`git commit -s`)
- [ ] New or changed packages meet the coverage floor in `.testcoverage.yml`
- [ ] A new device-payload decoder ships a fuzz target
- [ ] No gate was weakened to pass
