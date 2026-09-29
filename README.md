# tickerbox-cli

Command-line client for a TickerBox device's `/rest/` API.

## Build

```
go build -o tickerbox .
```

## Install

```
go install .
```

Installs `tickerbox` into `$(go env GOPATH)/bin`.

## Targeting a device

In order of precedence:

1. `--host URL`, e.g. `--host http://192.168.10.89`.
2. `--device NAME` (`-d`), a name registered with `device add` (see below).
3. `TICKERBOX_HOST` environment variable.
4. The default device set with `device use`.
5. `http://tickerbox.local`, resolved by the OS's mDNS resolver.

```
tickerbox device add office http://192.168.10.89
tickerbox device use office
tickerbox --device office wifi status
```

`device add` and `device use` only edit the local device config file (`tickerbox device list` shows its path); they make no request to the device.

## Global flags

| Flag | Effect |
| --- | --- |
| `--host URL` | Device base URL. Overrides `--device`, `$TICKERBOX_HOST` and the default device. |
| `-d, --device NAME` | Named device from the config file. |
| `-j, --json` | Emit raw device JSON instead of a decoded table. |
| `--retry N` | Retry an idempotent request up to N times on a transient failure (default 2). |
| `--timeout D` | Per-request timeout, e.g. `5s` (default 10s). |

These are persistent flags on the root command, so they go before the command group: `tickerbox --host http://10.0.0.5 --timeout 5s wifi status`.

## Command groups

- `status` — aggregate dashboard: features, system, network, AP, NTP, display and clock state in one call. `tickerbox status`, or `tickerbox status --all` for every configured device.
- `wifi` — station SSID, credentials, static IP, and network scan. `tickerbox wifi scan`
- `ap` — device's own access point. `tickerbox ap status`
- `ntp` — NTP sync configuration, including timezone. `tickerbox ntp set --tz "Europe/London"`
- `time` — set the device wall-clock time; with no argument, sends the current UTC instant. `tickerbox time 2026-01-01T00:00:00`
- `tz` — timezone lookup and device timezone. `tickerbox tz list --grep Europe`, `tickerbox tz set "Europe/London"`
- `display` — ticker screen brightness, rotation interval, sleep schedule. `tickerbox display settings`
- `clock` — clock screen. `tickerbox clock set --enabled`
- `brightness` — shortcut for display brightness: set, or step up/down. `tickerbox brightness 180`, `tickerbox brightness up 20`
- `uptime` — device uptime since last boot, formatted as `1d 2h 3m 4s`. `tickerbox uptime`
- `reboot` — shortcut for `system restart`, with a confirmation prompt. `tickerbox reboot -y`
- `tickers` — the asset list shown on the display:
  - `add` — append tickers sharing one type/time/currency. `tickerbox tickers add --type crypto --time 15min --currency EUR BTC ETH`
  - `edit` — change one entry in place, by index or symbol. `tickerbox tickers edit BTC --time 5min`
  - `move` — reorder the display cycle. `tickerbox tickers move 3 1`
  - `remove`, `clear`, `list`, `export`, `import`
  - `validate` — report illegal fields and duplicate symbols without changing anything. `tickerbox tickers validate`
  - `template` — built-in presets: `list`, `show <name>`, `apply <name>`. `tickerbox tickers template apply crypto-majors`
- `system` — device status, features, and maintenance actions. `tickerbox system info`; `system restart`, `system factory-reset` and `system firmware-upload <path>` mutate the device and prompt for confirmation unless `--yes`/`-y` is given.
- `config` — whole-device config as one JSON snapshot, covering every section (tickers, display, clock, ntp, wifi, ap). `tickerbox config export --file office.json`, `tickerbox config diff office.json`, `tickerbox config import office.json`
- `profile` — named, device-agnostic snapshots stored locally, default section set tickers/display/clock/ntp (`--all` or `--include` widens it). `tickerbox profile save office`, `tickerbox profile apply office`, `tickerbox profile diff office`
- `device` — the local device registry: `add NAME HOST`, `use NAME`, `list`, `rm NAME`, `ping [NAME|--all]`, `discover` (LAN scan for TickerBoxes). `tickerbox device discover`
- `doctor` — read-only health check (heap, filesystem headroom, wifi, NTP, AP exposure); exits non-zero on a critical finding. `tickerbox doctor --all`
- `watch` — re-runs a read-only view (`status` by default) on an interval until interrupted; refuses any command that is not read-only. `tickerbox watch tickers list --interval 5s`
- `version` — CLI build version. `tickerbox version`
- `open` (alias `ui`) — opens the device's web UI in the default browser. `tickerbox open`

## Endpoints covered

| Command | Method | Endpoint |
| --- | --- | --- |
| `status` | GET | `features`, `systemStatus`, `wifiStatus`, `apStatus`, `ntpStatus`, `settingsState`, `clockSetupState` |
| `wifi status` | GET | `wifiStatus` |
| `wifi settings` | GET | `wifiSettings` |
| `wifi set` | GET, POST | `wifiSettings` |
| `wifi scan` | GET | `scanNetworks`, `listNetworks` |
| `ap status` | GET | `apStatus` |
| `ap settings` | GET | `apSettings` |
| `ap set` | GET, POST | `apSettings` |
| `ntp status` | GET | `ntpStatus` |
| `ntp settings` | GET | `ntpSettings` |
| `ntp set` | GET, POST | `ntpSettings` |
| `time` | POST | `time` |
| `tz set` | GET, POST | `ntpSettings`, `clockSetupState` |
| `tz list` | — | bundled `timezones.json`, no device call |
| `display settings` | GET | `settingsState` |
| `display set` | GET, POST | `settingsState` |
| `brightness`, `brightness up`/`down` | GET, POST | `settingsState` |
| `clock settings` | GET | `clockSetupState` |
| `clock set` | GET, POST | `clockSetupState` |
| `tickers list`/`add`/`edit`/`move`/`remove`/`clear`/`export`/`import`/`validate`/`template apply` | GET, POST | `coinSetupState` |
| `tickers template list`/`show` | — | bundled presets, no device call |
| `system info` | GET | `systemStatus` |
| `system features` | GET | `features` |
| `system restart`, `reboot` | POST | `restart` |
| `system factory-reset` | POST | `factoryReset` |
| `system firmware-upload` | POST (multipart) | `uploadFirmware` |
| `uptime` | GET | `ntpStatus` |
| `config export`/`diff`/`import` | GET, POST | `coinSetupState`, `settingsState`, `clockSetupState`, `ntpSettings`, `wifiSettings`, `apSettings` |
| `profile save`/`apply`/`diff` | GET, POST | same section set as `config`, restricted to what was captured |
| `device discover`/`ping` | GET | `features` (probe only, to confirm a host is a TickerBox) |
| `doctor` | GET | `features`, `systemStatus`, `wifiStatus`, `apStatus`, `ntpStatus` |

`device add`, `device use`, `device list`, `device rm`, `profile list`/`show`/`rm`, `tz list` and `version` make no device call.

## Tests

```
go test ./...
```

Tests use `net/http/httptest`; they never contact a device.
