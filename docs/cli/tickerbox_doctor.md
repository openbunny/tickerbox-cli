## tickerbox doctor

Read-only health check

### Synopsis

Runs read-only checks: free heap, filesystem headroom, wifi connectivity, ntp sync, and ap exposure, plus a standing note that the device's REST API has no authentication. Exits non-zero if any check is critical. --all runs against every configured device.

```
tickerbox doctor [flags]
```

### Examples

```
  tickerbox doctor
  tickerbox doctor --all --json
```

### Options

```
      --all    run against every device configured in the device config file
  -h, --help   help for doctor
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

