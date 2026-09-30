## tickerbox fmp set-key

Store a Financial Modeling Prep API key in the config file

### Synopsis

Persists KEY to config.toml. $TICKERBOX_FMP_API_KEY, when set, overrides the stored key without changing it.

```
tickerbox fmp set-key <key> [flags]
```

### Examples

```
  tickerbox fmp set-key abc123
```

### Options

```
  -h, --help   help for set-key
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

* [tickerbox fmp](tickerbox_fmp.md)	 - Financial Modeling Prep API key, used to verify ticker symbols on add

