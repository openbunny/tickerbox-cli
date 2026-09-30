## tickerbox ticker rm

Remove a ticker

### Synopsis

Accepts either the 0-based index shown by `ticker list`, or a ticker symbol. Prompts for confirmation unless --yes.

```
tickerbox ticker rm <index|ticker> [flags]
```

### Examples

```
  tickerbox ticker rm BTC
  tickerbox ticker rm 0
```

### Options

```
  -h, --help   help for rm
  -y, --yes    skip confirmation
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

