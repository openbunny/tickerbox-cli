## tickerbox watch

Re-run a read-only view on an interval until interrupted

```
tickerbox watch [command args...] [flags]
```

### Options

```
  -h, --help                help for watch
      --interval duration   how often to refresh (default 2s)
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

