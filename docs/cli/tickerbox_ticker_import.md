## tickerbox ticker import

Replace tickers from a JSON file

### Synopsis

Replaces the entire ticker list with the contents of the file; existing entries not present in the file are dropped. Prompts for confirmation unless --yes.

```
tickerbox ticker import [flags]
```

### Examples

```
  tickerbox ticker import --input tickers.json --yes
```

### Options

```
  -h, --help           help for import
      --input string   JSON file with an array of ticker entries
  -y, --yes            skip confirmation
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

