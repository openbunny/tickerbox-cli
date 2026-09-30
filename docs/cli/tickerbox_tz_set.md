## tickerbox tz set

Set the device timezone

### Synopsis

label must be one shown by `tickerbox tz list`. Sets both the ntp and clock timezone fields on the device.

```
tickerbox tz set <label> [flags]
```

### Examples

```
  tickerbox tz set America/New_York
```

### Options

```
  -h, --help   help for set
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

* [tickerbox tz](tickerbox_tz.md)	 - Time zones

