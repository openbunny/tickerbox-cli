## tickerbox device current

Show the device the CLI currently targets

### Synopsis

Shows the host that resolves from --host, --device, $TICKERBOX_HOST, or the config default, in that order, and whether it matches the stored default device.

```
tickerbox device current [flags]
```

### Examples

```
  tickerbox device current
```

### Options

```
  -h, --help   help for current
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

* [tickerbox device](tickerbox_device.md)	 - Manage configured devices

