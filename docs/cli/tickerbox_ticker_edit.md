## tickerbox ticker edit

Change fields of one ticker entry in place

```
tickerbox ticker edit <index|sym> [flags]
```

### Examples

```
  tickerbox ticker edit BTC --time 1min
```

### Options

```
      --currency string   USD|EUR|GBP|CAD|AUD|JPY
  -h, --help              help for edit
      --ticker string     new ticker symbol
      --time string       1min|5min|15min
      --type string       crypto|stocks|forex
  -y, --yes               skip confirmation
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

