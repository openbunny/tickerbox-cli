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
