# Configuration schema

Two files live under the OS's per-user config directory
(`os.UserConfigDir()/tickerbox/`): the device config (`config.toml`,
this document's main subject) and, in a `profiles/` subdirectory, saved
device snapshots (`internal/profile`, JSON, schema at
[`schema/snapshot.schema.json`](schema/snapshot.schema.json)).

## Location

| OS      | `os.UserConfigDir()` resolves to         | Device config                                         |
| ------- | ---------------------------------------- | ----------------------------------------------------- |
| macOS   | `$HOME/Library/Application Support`      | `~/Library/Application Support/tickerbox/config.toml` |
| Linux   | `$XDG_CONFIG_HOME`, else `$HOME/.config` | `$XDG_CONFIG_HOME/tickerbox/config.toml`              |
| Windows | `%AppData%`                              | `%AppData%\tickerbox\config.toml`                     |

> [!NOTE]
> `internal/config.Path` cannot report an error: if `os.UserConfigDir`
> fails (no usable home directory), it falls back to the relative path
> `tickerbox/config.toml`, and `Load`/`Save` then fail with the
> underlying cause the first time that path is actually used.
> `internal/profile.Dir` fails differently on the same condition: it
> falls back to `os.TempDir()/tickerbox/profiles` instead of a relative
> path, so a profile store and the device config can end up in
> unrelated places on a system with no resolvable config directory.

## Permissions

The config directory is created `0700` and `config.toml` is written
`0600` (`internal/config.configDirPerm`, `configFilePerm`). `Load` warns
to stderr, but does not refuse to read, a file that is group- or
other-readable (any of mode bits `0044` set) — the config may hold a
device host on a private network but carries no credential itself; see
[Secret handling](#secret-handling) below for what does.

The profile store (`internal/profile.dirPerm`, `filePerm`) uses the same
`0700`/`0600` pair, with the same non-fatal stderr warning on a
group/other-readable profile file — a saved snapshot may carry wifi/ap
secrets, per below.

## `config.toml` schema

```toml
Default = "kitchen"
FMPAPIKey = "abc123"

[Devices.kitchen]
Name = "kitchen"
Host = "http://192.168.1.10"

[Devices.office]
Name = "office"
Host = "http://tickerbox-office.local"
```

| Field                 | TOML type | Go type             | Meaning                                                                                                                                                                                                                                              |
| --------------------- | --------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Default`             | string    | `string`            | Device name used when no `--host`/`--device` flag and no `$TICKERBOX_HOST` are set. Empty string means no default. Set automatically to the first device's name when `device add` runs and no default is set yet; `device use` changes it afterward. |
| `Devices`             | table     | `map[string]Device` | Keyed by device name; the key and `Devices.<key>.Name` are kept equal by every write path (`Config.Add`).                                                                                                                                            |
| `Devices.<name>.Name` | string    | `string`            | Same value as the table key.                                                                                                                                                                                                                         |
| `Devices.<name>.Host` | string    | `string`            | Base URL the CLI sends `/rest/...` requests against, e.g. `http://192.168.1.10` or a `.local` mDNS name. Not validated as a URL by `Config.Add`.                                                                                                     |
| `FMPAPIKey`           | string    | `string`            | Financial Modeling Prep API key, set by `tickerbox fmp set-key` and used to verify a ticker symbol on `ticker add`. Omitted from the file when empty. `$TICKERBOX_FMP_API_KEY` takes precedence when set; see [Secret handling](#secret-handling).   |

`device add` does not add a scheme for you: `Config.Add` and the request
path (`cmd.newClient()` → `restBase(host)`) both use the host exactly as
given. Only `device discover`'s auto-add flow prepends `http://`. A bare
hostname such as `tickerbox-office.local`, added directly with `device add`,
fails at request time with `unsupported protocol scheme ""` — give `Host` a
scheme, as in the example above.

Field casing is exact: `pelletier/go-toml/v2` marshals unadorned struct
fields under their Go names, so the keys are `Default`/`Devices`/`Name`/`Host`,
not lowercased.

A missing `config.toml` is not an error: `Load` returns an empty
`Config{Devices: map[string]Device{}}`, since a first-run install has no
config yet.

## Host resolution order

`Config.Resolve(hostFlag, deviceFlag string)`, called from every
device-targeting command's `PersistentPreRunE` in `cmd/root.go`, picks the
first of:

1. `hostFlag` (`--host`), if non-empty.
2. `Devices[deviceFlag].Host` (`--device`/`-d`), if `deviceFlag` is
   non-empty — an unknown device name is an error, not a fallthrough.
3. `$TICKERBOX_HOST`.
4. `Devices[Default].Host`, if `Default` is set and names a device that
   still exists.

With none of the above set, resolution fails with an error naming three
fixes: `device add`, `device use`, or `--host`.

## Secret handling

`config.toml` never holds a device credential. It does hold one CLI-level
secret: `FMPAPIKey`, set by `tickerbox fmp set-key` and read by `ticker add`'s
Financial Modeling Prep symbol verification. It has no device-facing use — the
device never sees it — and is not a wifi/ap password, so it falls outside
`internal/section`'s password/secret field stripping described below.
`$TICKERBOX_FMP_API_KEY`, when set, is used instead of the stored value without
changing it. No command prints `FMPAPIKey` in the clear; wherever it appears in
output it is masked the same way a wifi/ap password is.

By default,
`internal/section.Capture` strips any wifi or ap field whose key contains
`password` or `secret` (case-insensitive substring, not a fixed field list)
before it reaches a Snapshot, so no command writes a device secret to a
local file unless told to.

Two commands write a secret into a file: `tickerbox profile save --all`
captures the wifi/ap sections, including their passwords, into the profile
file, and `tickerbox config export --show-secrets` does the same into the
exported snapshot file. See
[`schema/snapshot.schema.json`](schema/snapshot.schema.json) for the export
file's schema (top level) and the profile file's schema
(`$defs/profileRecord`).

A profile file on disk is a superset of the snapshot shape:
`{"description": <string, omitted if empty>, "source_device": <string,
omitted if empty>, ...section.Snapshot fields}`. `description` comes from
`profile save --description`; `source_device` is captured automatically from
the device the profile was saved from.

`tickerbox wifi settings --show-secrets` is different: it only skips
masking the password before printing the device's current, live response.
It never writes a Snapshot or any other file.
