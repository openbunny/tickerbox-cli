## tickerbox fmp

Financial Modeling Prep API key, used to verify ticker symbols on add

### Synopsis

Manages the Financial Modeling Prep API key `tickers add` uses to verify a symbol before adding it. Resolved from $TICKERBOX_FMP_API_KEY first, then the key stored here.

### Options

```
  -h, --help   help for fmp
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
* [tickerbox fmp set-key](tickerbox_fmp_set-key.md)	 - Store a Financial Modeling Prep API key in the config file

