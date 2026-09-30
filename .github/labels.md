# Label taxonomy

Three prefixes: `type/*` classifies an issue, `area/*` locates it, `status/*`
tracks it through triage. Every issue and PR carries exactly one `type/*`
label; `area/*` and `status/*` are added during triage.

## Contents

- [type/\*](#type)
- [area/\*](#area)
- [status/\*](#status)

## type/*

| Label           | Meaning                                               |
| --------------- | ----------------------------------------------------- |
| `type/bug`      | A command behaves differently from what it documents. |
| `type/feature`  | A new command, flag, or behaviour.                    |
| `type/docs`     | README, CONTRIBUTING, or other documentation only.    |
| `type/chore`    | Build, dependency, or repository maintenance.         |
| `type/security` | A vulnerability report or hardening change.           |

## area/*

| Label              | Command groups covered                                   |
| ------------------ | -------------------------------------------------------- |
| `area/wifi`        | `wifi`, `ap`                                             |
| `area/network`     | `ntp`, `time`, `tz`                                      |
| `area/display`     | `display`, `clock`, `brightness`                         |
| `area/tickers`     | `tickers`                                                |
| `area/system`      | `system` (including firmware upload), `reboot`           |
| `area/config`      | `config`, `profile`                                      |
| `area/device`      | `device` (local device registry)                         |
| `area/diagnostics` | `status`, `doctor`, `watch`, `uptime`, `version`, `open` |
| `area/ci`          | GitHub Actions workflows, `.goreleaser.yaml`             |
| `area/docs`        | README, CONTRIBUTING, and other tracked docs             |

## status/*

| Label                 | Meaning                                                          |
| --------------------- | ---------------------------------------------------------------- |
| `status/needs-triage` | Default label on a new issue; a maintainer has not reviewed it.  |
| `status/confirmed`    | A maintainer reproduced the bug or accepted the proposal.        |
| `status/blocked`      | Waiting on an external dependency, decision, or device firmware. |
| `status/wontfix`      | Closed without a change; the issue states why.                   |
