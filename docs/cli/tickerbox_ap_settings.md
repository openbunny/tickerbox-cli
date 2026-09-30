## tickerbox ap settings

Show access point settings

### Synopsis

The AP password is always shown in the clear; there is no masking flag.

```
tickerbox ap settings [flags]
```

### Examples

```
  tickerbox ap settings
```

### Options

```
  -h, --help   help for settings
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

* [tickerbox ap](tickerbox_ap.md)	 - Access point

