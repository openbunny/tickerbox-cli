# Security policy

## Reporting a vulnerability

Report suspected vulnerabilities privately, not through a public issue. Where
GitHub private vulnerability reporting is enabled, use the repository's
**Security** tab and choose **Report a vulnerability**; otherwise contact an
organization administrator directly.

Include the affected command or endpoint, the impact, and steps to reproduce. A
maintainer acknowledges the report and coordinates a fix and disclosure.

## Scope

This policy covers the `tickerbox` CLI in this repository. It does not cover the
TickerBox device firmware.

The device's REST API serves the Wi-Fi and access-point passwords in cleartext
to any unauthenticated client on its network (`GET /rest/wifiSettings`,
`GET /rest/apSettings`). That is a device-firmware exposure, not a defect in
this CLI; `tickerbox doctor` reports it.

## Network behaviour

`tickerbox` makes network calls only to the device host resolved from
`--host`, `--device`, `$TICKERBOX_HOST`, or the default device set with
`device use`. It never contacts any host outside that resolution, sends no
telemetry, and does not check for updates on its own.

## Verifying releases

Release checksums are signed with a project-controlled cosign key. Verifying
the signature proves `checksums.txt` was signed by whoever holds the private
key matching the public key below — a key-based signature carries no
embedded identity, so it does not itself assert who that is. The matching
private key is held only by release CI; the project's public statement is
that this is the same identity, `bot@fiona.sm`, that OpenPGP-signs every
commit.

Public key:

```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEO2QgVQB5kqIixdo9dkl8LDlmqqb2
sSsTp/18FWNMfZQTVlYqQkNrurSX4J95mFYf40x7ht4hjs96wOWjt/7Erg==
-----END PUBLIC KEY-----
```

Save the key as `cosign.pub`, then verify a downloaded release:

```console
cosign verify-blob --key cosign.pub --bundle checksums.txt.bundle --insecure-ignore-tlog checksums.txt
sha256sum --ignore-missing --check checksums.txt
```

`checksums.txt` lists every archive and package built for the release, not
only the one you downloaded; `--ignore-missing` skips lines for files not
present in the working directory instead of reporting them as failures.

On macOS, which ships `shasum` rather than GNU coreutils' `sha256sum`, use:

```console
shasum -a 256 --ignore-missing --check checksums.txt
```

Each release archive also ships a CycloneDX SBOM, covered by the same
`checksums.txt`/cosign verification chain as the archive itself.
