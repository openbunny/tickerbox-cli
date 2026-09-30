## tickerbox ticker add

Add one or more tickers, sharing --type/--time/--currency

### Synopsis

Adds one entry per sym, all sharing the same --type, --time, and --currency. --type and --time are required; --currency defaults to USD.

```
tickerbox ticker add <sym>... [flags]
```

### Examples

```
  tickerbox ticker add BTC ETH --type crypto --time 5min --currency USD
```

### Options

```
      --currency string   USD|EUR|GBP|CAD|AUD|JPY (default "USD")
  -h, --help              help for add
      --no-verify         skip Financial Modeling Prep symbol verification, add the entry unchecked
      --time string       1min|5min|15min
      --type string       crypto|stocks|forex
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

