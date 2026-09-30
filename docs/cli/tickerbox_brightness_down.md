## tickerbox brightness down

Decrease display brightness, clamped to 10-255

### Synopsis

Lowers brightness by step, or by 25 if step is omitted, clamped to 10-255.

```
tickerbox brightness down [step] [flags]
```

### Examples

```
  tickerbox brightness down
```

### Options

```
  -h, --help   help for down
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

* [tickerbox brightness](tickerbox_brightness.md)	 - Set display brightness

