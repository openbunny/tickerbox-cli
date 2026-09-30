## tickerbox device discover

Find TickerBoxes on the local network

### Synopsis

Probes tickerbox.local, then scans every host on the local /24 subnet for a TickerBox. For each one found, prompts interactively to add it as a named device.

```
tickerbox device discover [flags]
```

### Examples

```
  tickerbox device discover
```

### Options

```
  -h, --help   help for discover
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

* [tickerbox device](tickerbox_device.md)	 - Manage configured devices

