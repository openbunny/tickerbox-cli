## tickerbox ticker

Ticker / asset list

### Options

```
  -h, --help   help for ticker
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

* [tickerbox](tickerbox.md)	 - Control a TickerBox device over its REST API
* [tickerbox ticker add](tickerbox_ticker_add.md)	 - Add one or more tickers, sharing --type/--time/--currency
* [tickerbox ticker clear](tickerbox_ticker_clear.md)	 - Remove all tickers
* [tickerbox ticker edit](tickerbox_ticker_edit.md)	 - Change fields of one ticker entry in place
* [tickerbox ticker export](tickerbox_ticker_export.md)	 - Export tickers as JSON
* [tickerbox ticker import](tickerbox_ticker_import.md)	 - Replace tickers from a JSON file
* [tickerbox ticker list](tickerbox_ticker_list.md)	 - List configured tickers
* [tickerbox ticker move](tickerbox_ticker_move.md)	 - Reorder the ticker list; the display cycles in this order
* [tickerbox ticker rm](tickerbox_ticker_rm.md)	 - Remove a ticker
* [tickerbox ticker template](tickerbox_ticker_template.md)	 - Built-in ticker list presets
* [tickerbox ticker validate](tickerbox_ticker_validate.md)	 - Report illegal fields and duplicate symbols in the ticker list

