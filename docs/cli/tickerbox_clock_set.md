## tickerbox clock set

Change clock settings

```
tickerbox clock set [flags]
```

### Options

```
      --12h                   use 12-hour time format
      --24h                   use 24-hour time format
      --animation-speed int   clock animation speed (10-200)
      --disabled              disable the clock screen
      --enabled               enable the clock screen
  -h, --help                  help for set
      --tz string             timezone label
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

* [tickerbox clock](tickerbox_clock.md)	 - Clock screen settings

