# tickerbox-cli

Command-line client for a TickerBox device's `/rest/` API.

[![CI](https://github.com/openbunny/tickerbox-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/openbunny/tickerbox-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/openbunny/tickerbox-cli.svg)](https://pkg.go.dev/github.com/openbunny/tickerbox-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/openbunny/tickerbox-cli)](https://goreportcard.com/report/github.com/openbunny/tickerbox-cli)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/openbunny/tickerbox-cli/badge)](https://securityscorecards.dev/viewer/?uri=github.com/openbunny/tickerbox-cli)

## Table of contents

- [Quickstart](#quickstart)
- [Install](#install)
- [Targeting a device](#targeting-a-device)
- [Global flags](#global-flags)
- [Command groups](#command-groups)
- [Endpoints covered](#endpoints-covered)
- [Shell completion](#shell-completion)
- [Network behaviour](#network-behaviour)
- [Tests](#tests)
- [Documentation](#documentation)

## Quickstart

**1. Find the device.** `device discover` probes `tickerbox.local` and scans
the local `/24` subnet for anything that answers as a TickerBox, then offers
to save each match:

```console
$ tickerbox device discover
HOST            HOSTNAME
192.168.10.89   tickerbox
Add 192.168.10.89 as a device? [y/N]: y
Name for 192.168.10.89: office
added device office (http://192.168.10.89)
```

**2. Check it.** `status` pulls every dashboard endpoint in one call:

```console
$ tickerbox --device office status
Features
Project:            yes
NTP:                yes
OTA:                yes
Upload Firmware:    yes
TickerBox:          yes
Shopify:            no
Extra ETF:          no

System
Platform:          esp32
SDK Version:       v5.1.2
CPU Frequency:     240 MHz
Free Heap:         142.3 KB
Flash Chip Size:   4.0 MB
Filesystem:        612.0 KB / 1.5 MB used

Wi-Fi
Status:        CONNECTED
SSID:          HomeNet
IP Address:    192.168.10.89
RSSI:          -52 dBm
Channel:       6
...
```

**3. Add tickers.** `tickers add` appends one or more symbols that share a
type, refresh interval and currency:

```console
$ tickerbox --device office tickers add --type crypto --time 15min --currency EUR BTC ETH
Added 2 ticker(s)
```

Set `office` as the default so `--device office` can be dropped from later
commands:

```console
tickerbox device use office
```

## Install

```console
$ go install github.com/openbunny/tickerbox-cli@latest
```

`go install` places the `tickerbox` binary in `$(go env GOPATH)/bin`. Prebuilt
binaries, `.deb` and `.rpm` packages are attached to each
[release](https://github.com/openbunny/tickerbox-cli/releases).

From a clone:

```console
$ go build -o tickerbox .
```

## Targeting a device

`tickerbox` resolves the device host once per invocation, in this order:

| Precedence | Source                                                       | Example                                                |
| ---------- | ------------------------------------------------------------ | ------------------------------------------------------ |
| 1          | `--host URL`                                                 | `--host http://192.168.10.89`                          |
| 2          | `--device NAME` (`-d`)                                       | `--device office`, a name registered with `device add` |
| 3          | `TICKERBOX_HOST` environment variable                        | `TICKERBOX_HOST=http://192.168.10.89 tickerbox status` |
| 4          | The default device set with `device use`                     | `tickerbox device use office`                          |
| 5          | `http://tickerbox.local`, resolved by the OS's mDNS resolver | no flag, no env var, no default set                    |

The first source with a value wins; lower rows are never consulted once a
higher one resolves.

```console
$ tickerbox device add office http://192.168.10.89
added device office (http://192.168.10.89)
$ tickerbox device use office
default device set to office
$ tickerbox --device office wifi status
```

> **Note**
> `device add` and `device use` only edit the local device config file
> (`tickerbox device list` shows its path); neither makes a request to the
> device.

## Global flags

These are persistent flags on the root command, so they go before the command
group: `tickerbox --host http://10.0.0.5 --timeout 5s wifi status`.

| Flag                | Effect                                                                           |
| ------------------- | -------------------------------------------------------------------------------- |
| `--host URL`        | Device base URL. Overrides `--device`, `$TICKERBOX_HOST` and the default device. |
| `-d, --device NAME` | Named device from the config file.                                               |
| `-j, --json`        | Emit raw device JSON instead of a decoded table.                                 |
| `--retry N`         | Retry an idempotent request up to N times on a transient failure (default 2).    |
| `--timeout D`       | Per-request timeout, e.g. `5s` (default 10s).                                    |

## Command groups

| Group               | Purpose                                                                                                                                                                        | Example                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------- |
| `status`            | Aggregate dashboard: features, system, network, AP, NTP, display and clock state in one call.                                                                                  | `tickerbox status`, or `--all` for every configured device                |
| `wifi`              | Station SSID, credentials, static IP, and network scan.                                                                                                                        | `tickerbox wifi scan`                                                     |
| `ap`                | The device's own access point.                                                                                                                                                 | `tickerbox ap status`                                                     |
| `ntp`               | NTP sync configuration, including timezone.                                                                                                                                    | `tickerbox ntp set --tz "Europe/London"`                                  |
| `time`              | Set the device wall-clock time; with no argument, sends the current UTC instant.                                                                                               | `tickerbox time 2026-01-01T00:00:00`                                      |
| `tz`                | Timezone lookup and device timezone.                                                                                                                                           | `tickerbox tz list --grep Europe`, `tickerbox tz set "Europe/London"`     |
| `display`           | Ticker screen brightness, rotation interval, sleep schedule.                                                                                                                   | `tickerbox display settings`                                              |
| `clock`             | Clock screen.                                                                                                                                                                  | `tickerbox clock set --enabled`                                           |
| `brightness`        | Shortcut for display brightness: set, or step up/down.                                                                                                                         | `tickerbox brightness 180`, `tickerbox brightness up 20`                  |
| `uptime`            | Device uptime since last boot, formatted as `1d 2h 3m 4s`.                                                                                                                     | `tickerbox uptime`                                                        |
| `reboot`            | Shortcut for `system restart`, with a confirmation prompt.                                                                                                                     | `tickerbox reboot -y`                                                     |
| `tickers`           | The asset list shown on the display: `add`, `edit`, `move`, `remove`, `clear`, `list`, `export`, `import`, `validate`, `template`.                                             | `tickerbox tickers add --type crypto --time 15min --currency EUR BTC ETH` |
| `system`            | Device status, features, and maintenance actions. `restart`, `factory-reset` and `firmware-upload` mutate the device and prompt for confirmation unless `--yes`/`-y` is given. | `tickerbox system info`                                                   |
| `config`            | Whole-device config as one JSON snapshot, covering every section (tickers, display, clock, ntp, wifi, ap).                                                                     | `tickerbox config export --file office.json`                              |
| `profile`           | Named, device-agnostic snapshots stored locally; default section set is tickers/display/clock/ntp (`--all` or `--include` widens it).                                          | `tickerbox profile save office`                                           |
| `device`            | The local device registry: `add NAME HOST`, `use NAME`, `list`, `rm NAME`, `ping [NAME\|--all]`, `discover` (LAN scan for TickerBoxes).                                        | `tickerbox device discover`                                               |
| `doctor`            | Read-only health check (heap, filesystem headroom, wifi, NTP, AP exposure); exits non-zero on a critical finding.                                                              | `tickerbox doctor --all`                                                  |
| `watch`             | Re-runs a read-only view (`status` by default) on an interval until interrupted; refuses any command that is not read-only.                                                    | `tickerbox watch tickers list --interval 5s`                              |
| `version`           | CLI build version.                                                                                                                                                             | `tickerbox version`                                                       |
| `open` (alias `ui`) | Opens the device's web UI in the default browser.                                                                                                                              | `tickerbox open`                                                          |

`tickers add` (SYM... form shown above) merges what was formerly a
single-ticker `add`; every symbol given shares one `--type`/`--time`/
`--currency`.

### `tickers` subcommands

| Subcommand                                       | Effect                                                                 | Example                                                                   |
| ------------------------------------------------ | ---------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `add SYM...`                                     | Append tickers sharing one type/time/currency.                         | `tickerbox tickers add --type crypto --time 15min --currency EUR BTC ETH` |
| `edit <index\|sym>`                              | Change one entry in place, by index or symbol.                         | `tickerbox tickers edit BTC --time 5min`                                  |
| `move <from> <to>`                               | Reorder the display cycle.                                             | `tickerbox tickers move 3 1`                                              |
| `remove <index\|sym>`                            | Remove one entry.                                                      | `tickerbox tickers remove BTC`                                            |
| `clear`                                          | Remove every entry.                                                    | `tickerbox tickers clear`                                                 |
| `list`                                           | List configured tickers.                                               | `tickerbox tickers list`                                                  |
| `export` / `import`                              | Round-trip the list as JSON.                                           | `tickerbox tickers export --file list.json`                               |
| `validate`                                       | Report illegal fields and duplicate symbols without changing anything. | `tickerbox tickers validate`                                              |
| `template list` / `show <name>` / `apply <name>` | Built-in presets.                                                      | `tickerbox tickers template apply crypto-majors`                          |

## Endpoints covered

| Command                                                                                           | Method           | Endpoint                                                                                              |
| ------------------------------------------------------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------- |
| `status`                                                                                          | GET              | `features`, `systemStatus`, `wifiStatus`, `apStatus`, `ntpStatus`, `settingsState`, `clockSetupState` |
| `wifi status`                                                                                     | GET              | `wifiStatus`                                                                                          |
| `wifi settings`                                                                                   | GET              | `wifiSettings`                                                                                        |
| `wifi set`                                                                                        | GET, POST        | `wifiSettings`                                                                                        |
| `wifi scan`                                                                                       | GET              | `scanNetworks`, `listNetworks`                                                                        |
| `ap status`                                                                                       | GET              | `apStatus`                                                                                            |
| `ap settings`                                                                                     | GET              | `apSettings`                                                                                          |
| `ap set`                                                                                          | GET, POST        | `apSettings`                                                                                          |
| `ntp status`                                                                                      | GET              | `ntpStatus`                                                                                           |
| `ntp settings`                                                                                    | GET              | `ntpSettings`                                                                                         |
| `ntp set`                                                                                         | GET, POST        | `ntpSettings`                                                                                         |
| `time`                                                                                            | POST             | `time`                                                                                                |
| `tz set`                                                                                          | GET, POST        | `ntpSettings`, `clockSetupState`                                                                      |
| `tz list`                                                                                         | —                | bundled `timezones.json`, no device call                                                              |
| `display settings`                                                                                | GET              | `settingsState`                                                                                       |
| `display set`                                                                                     | GET, POST        | `settingsState`                                                                                       |
| `brightness`, `brightness up`/`down`                                                              | GET, POST        | `settingsState`                                                                                       |
| `clock settings`                                                                                  | GET              | `clockSetupState`                                                                                     |
| `clock set`                                                                                       | GET, POST        | `clockSetupState`                                                                                     |
| `tickers list`/`add`/`edit`/`move`/`remove`/`clear`/`export`/`import`/`validate`/`template apply` | GET, POST        | `coinSetupState`                                                                                      |
| `tickers template list`/`show`                                                                    | —                | bundled presets, no device call                                                                       |
| `system info`                                                                                     | GET              | `systemStatus`                                                                                        |
| `system features`                                                                                 | GET              | `features`                                                                                            |
| `system restart`, `reboot`                                                                        | POST             | `restart`                                                                                             |
| `system factory-reset`                                                                            | POST             | `factoryReset`                                                                                        |
| `system firmware-upload`                                                                          | POST (multipart) | `uploadFirmware`                                                                                      |
| `uptime`                                                                                          | GET              | `ntpStatus`                                                                                           |
| `config export`/`diff`/`import`                                                                   | GET, POST        | `coinSetupState`, `settingsState`, `clockSetupState`, `ntpSettings`, `wifiSettings`, `apSettings`     |
| `profile save`/`apply`/`diff`                                                                     | GET, POST        | same section set as `config`, restricted to what was captured                                         |
| `device discover`/`ping`                                                                          | GET              | `features` (probe only, to confirm a host is a TickerBox)                                             |
| `doctor`                                                                                          | GET              | `features`, `systemStatus`, `wifiStatus`, `apStatus`, `ntpStatus`                                     |

`device add`, `device use`, `device list`, `device rm`, `profile list`/`show`/`rm`, `tz list` and `version` make no device call.

## Shell completion

`tickerbox completion` generates a shell completion script; it is a cobra
built-in, not a project-specific feature, so its behaviour follows cobra's
current documentation for the subcommand of your shell.

| Shell      | Load for this session                                                | Install for every session                                                                                                                                                                                |
| ---------- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| bash       | `source <(tickerbox completion bash)`                                | `tickerbox completion bash > /etc/bash_completion.d/tickerbox` (Linux) or `tickerbox completion bash > $(brew --prefix)/etc/bash_completion.d/tickerbox` (macOS); requires the `bash-completion` package |
| zsh        | `source <(tickerbox completion zsh)`                                 | `tickerbox completion zsh > "${fpath[1]}/_tickerbox"`; requires `compinit` enabled in `~/.zshrc`                                                                                                         |
| fish       | `tickerbox completion fish \| source`                                | `tickerbox completion fish > ~/.config/fish/completions/tickerbox.fish`                                                                                                                                  |
| powershell | `tickerbox completion powershell \| Out-String \| Invoke-Expression` | add that line to your PowerShell profile                                                                                                                                                                 |

```console
$ tickerbox completion bash --help
Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(tickerbox completion bash)
...
```

## Network behaviour

`tickerbox` makes network calls only to the device host resolved from
`--host`, `--device`, `$TICKERBOX_HOST`, or the default device (see
[Targeting a device](#targeting-a-device)). It never contacts any host
outside that resolution and does not phone home.

Two exceptions send LAN broadcasts instead of a request to the resolved host:
`device discover` probes `tickerbox.local` and every address in the local
`/24` subnet, and `device ping` probes each configured device, both looking
only for TickerBoxes on the local network.

`tickerbox` does not check for updates on its own. `version` reports the
build embedded at compile time and makes no request.

## Tests

```console
$ go test ./...
ok  	github.com/openbunny/tickerbox-cli/cmd	0.412s
ok  	github.com/openbunny/tickerbox-cli/internal/client	0.038s
ok  	github.com/openbunny/tickerbox-cli/internal/config	0.021s
...
```

Tests use `net/http/httptest`; they never contact a device.

## Documentation

- [CLI reference](docs/cli/tickerbox.md) — every command and flag, generated from the cobra command tree.
- [Configuration schema](docs/config-schema.md) — the on-disk device config and profile store: location, permissions, field reference.
- [docs/README.md](docs/README.md) — index of the above plus the profile-snapshot JSON schema and the timezone-data provenance record.
