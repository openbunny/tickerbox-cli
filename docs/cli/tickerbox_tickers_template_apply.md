## tickerbox tickers template apply

Apply a preset to the current ticker list

```
tickerbox tickers template apply <name> [flags]
```

### Options

```
      --append    append to the current list, skipping duplicates (default) (default true)
  -h, --help      help for apply
      --replace   replace the current list instead of appending
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

* [tickerbox tickers template](tickerbox_tickers_template.md)	 - Built-in ticker list presets

