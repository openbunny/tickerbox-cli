## tickerbox ticker export

Export tickers as JSON

```
tickerbox ticker export [flags]
```

### Examples

```
  tickerbox ticker export --output tickers.json
```

### Options

```
  -h, --help            help for export
      --output string   file to write; defaults to stdout
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

* [tickerbox ticker](tickerbox_ticker.md)	 - Ticker / asset list

