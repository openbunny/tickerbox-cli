## tickerbox tickers

Ticker / asset list

### Options

```
  -h, --help   help for tickers
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
* [tickerbox tickers add](tickerbox_tickers_add.md)	 - Add one or more tickers, sharing --type/--time/--currency
* [tickerbox tickers clear](tickerbox_tickers_clear.md)	 - Remove all tickers
* [tickerbox tickers edit](tickerbox_tickers_edit.md)	 - Change fields of one ticker entry in place
* [tickerbox tickers export](tickerbox_tickers_export.md)	 - Export tickers as JSON
* [tickerbox tickers import](tickerbox_tickers_import.md)	 - Replace tickers from a JSON file
* [tickerbox tickers list](tickerbox_tickers_list.md)	 - List configured tickers
* [tickerbox tickers move](tickerbox_tickers_move.md)	 - Reorder the ticker list; the display cycles in this order
* [tickerbox tickers remove](tickerbox_tickers_remove.md)	 - Remove a ticker
* [tickerbox tickers template](tickerbox_tickers_template.md)	 - Built-in ticker list presets
* [tickerbox tickers validate](tickerbox_tickers_validate.md)	 - Report illegal fields and duplicate symbols in the ticker list

