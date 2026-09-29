## tickerbox display set

Change display settings

```
tickerbox display set [flags]
```

### Options

```
      --brightness int       display brightness (10-255)
  -h, --help                 help for set
      --interval int         seconds between screens (10-60)
      --no-sleep             disable scheduled sleep
      --sleep                enable scheduled sleep
      --sleep-end string     sleep end time, ISO8601
      --sleep-start string   sleep start time, ISO8601
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

* [tickerbox display](tickerbox_display.md)	 - Display brightness / rotation / sleep

