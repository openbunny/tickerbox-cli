## tickerbox brightness

Set display brightness

### Synopsis

Sets brightness directly to a value from 10-255. Use the up/down subcommands to step relative to the current value instead.

```
tickerbox brightness [10-255] [flags]
```

### Examples

```
  tickerbox brightness 180
```

### Options

```
  -h, --help   help for brightness
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
* [tickerbox brightness down](tickerbox_brightness_down.md)	 - Decrease display brightness, clamped to 10-255
* [tickerbox brightness up](tickerbox_brightness_up.md)	 - Increase display brightness, clamped to 10-255

