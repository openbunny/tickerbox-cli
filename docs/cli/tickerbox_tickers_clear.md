## tickerbox tickers clear

Remove all tickers

### Synopsis

Removes every ticker. Prompts for confirmation unless --yes.

```
tickerbox tickers clear [flags]
```

### Examples

```
  tickerbox tickers clear --yes
```

### Options

```
  -h, --help   help for clear
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

* [tickerbox tickers](tickerbox_tickers.md)	 - Ticker / asset list

