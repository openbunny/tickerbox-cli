## tickerbox tickers move

Reorder the ticker list; the display cycles in this order

### Synopsis

FROM and TO are 0-based positions in the order the display cycles through, as shown by `tickers list`.

```
tickerbox tickers move <from> <to> [flags]
```

### Examples

```
  tickerbox tickers move 3 0
```

### Options

```
  -h, --help   help for move
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

* [tickerbox tickers](tickerbox_tickers.md)	 - Ticker / asset list

