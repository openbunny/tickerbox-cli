## tickerbox wifi scan

Scan for nearby Wi-Fi networks

### Synopsis

Triggers a scan and polls for results, retrying briefly if none are ready yet. Results are sorted by signal strength, strongest first.

```
tickerbox wifi scan [flags]
```

### Examples

```
  tickerbox wifi scan
```

### Options

```
  -h, --help   help for scan
```

### Options inherited from parent commands

```
  -d, --device string      named device from the config file
      --host string        TickerBox base URL ($TICKERBOX_HOST, or the default device; error if neither is set)
  -j, --json               emit raw device JSON instead of a decoded table
      --retry int          retry an idempotent request this many times on a transient failure (default 2)
      --timeout duration   per-request timeout (default 10s)
```

### SEE ALSO

* [tickerbox wifi](tickerbox_wifi.md)	 - Wi-Fi station configuration

