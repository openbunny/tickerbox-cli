# Provenance: internal/tz/timezones.json

`timezones.json` maps each IANA time zone label (e.g. `Africa/Cairo`) to its
POSIX TZ string, read from the trailing footer of the corresponding compiled
zoneinfo file (TZif format, footer defined in
[RFC 8536 §3.3](https://www.rfc-editor.org/rfc/rfc8536#section-3.3)).

Source authority: the [IANA Time Zone Database](https://www.iana.org/time-zones).
Its own licensing statement, shipped in each release as `LICENSE`, is at
<https://data.iana.org/time-zones/releases/>.

## Current file

Committed in this repository's initial commit. The upstream tzdata release
and capture date used to produce it are not recorded and are unknown as of
this document.

## Regeneration

`internal/tz/gen/main.go` (invoked via `//go:generate go run gen/main.go` in
`tz.go`, run as `go generate ./internal/tz`) downloads a dated tzdata release
from `https://data.iana.org/time-zones/releases/`, compiles it with the
system `zic` binary (required on `PATH`), reads the POSIX TZ footer out of
each compiled zone file, and writes the result to `timezones.json`.

The release it targets by default is the `pinnedRelease` constant in
`gen/main.go`, currently `2025b`. Before running, confirm this is still the
latest release at <https://www.iana.org/time-zones>; update the constant (and
this document's pinned-release line) first if not.

This environment cannot reach the network, so the generator has not been run
here and `timezones.json` is unchanged. After a successful run elsewhere,
replace this section with the release actually fetched, its file date from
the releases index, and the UTC time of the run.
