## tickerbox ticker validate

Report illegal fields and duplicate symbols in the ticker list

```
tickerbox ticker validate [flags]
```

### Examples

```
  tickerbox ticker validate
```

### Options

```
  -h, --help   help for validate
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

* [tickerbox ticker](tickerbox_ticker.md)	 - Ticker / asset list

