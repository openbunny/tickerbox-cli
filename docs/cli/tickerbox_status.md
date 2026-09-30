## tickerbox status

Aggregate dashboard

### Synopsis

Shows features, system, Wi-Fi, AP, NTP, display, and clock state in one report. --all runs this against every device in the config file instead of the resolved target.

```
tickerbox status [flags]
```

### Examples

```
  tickerbox status --all --json
```

### Options

```
      --all    show status for every device in the config file instead of the resolved target
  -h, --help   help for status
```

### Options inherited from parent commands

```
  -d, --device string      named device from the config file
      --host string        TickerBox base URL (default http://tickerbox.local, or $TICKERBOX_HOST)
  -j, --json               emit raw device JSON instead of a decoded table
      --retry int          retry an idempotent request this many times on a transient failure (default 2)
      --timeout duration   per-request timeout (default 10s)
```

### SEE ALSO

* [tickerbox](tickerbox.md)	 - Control a TickerBox device over its REST API

