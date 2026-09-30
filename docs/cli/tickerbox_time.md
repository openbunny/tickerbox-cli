## tickerbox time

Device wall-clock time

### Synopsis

Sets the device wall clock. With no value, or value `now`, uses the current UTC time. Otherwise value must be RFC3339 or 2006-01-02T15:04:05, and is sent to the device as UTC.

```
tickerbox time [value] [flags]
```

### Examples

```
  tickerbox time
  tickerbox time 2026-01-15T09:00:00Z
```

### Options

```
  -h, --help   help for time
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

