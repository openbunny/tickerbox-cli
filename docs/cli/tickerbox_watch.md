## tickerbox watch

Re-run a read-only view on an interval until interrupted

### Synopsis

Re-runs a read-only view on an interval until interrupted, clearing the screen each time. Only a fixed set of read-only commands can be targeted (status, system info/features, wifi/ap/ntp status and settings, display/clock settings, tz list, ticker list); with no arguments it re-runs status.

```
tickerbox watch [command args...] [flags]
```

### Examples

```
  tickerbox watch
  tickerbox watch --interval 5s wifi status
```

### Options

```
  -h, --help                help for watch
      --interval duration   how often to refresh (default 2s)
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

* [tickerbox](tickerbox.md)	 - Control a TickerBox device over its REST API

