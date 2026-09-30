## tickerbox tz

Time zones

### Options

```
  -h, --help   help for tz
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
* [tickerbox tz list](tickerbox_tz_list.md)	 - List known time zone labels and their POSIX strings
* [tickerbox tz set](tickerbox_tz_set.md)	 - Set the device timezone

