## tickerbox device list

List configured devices

```
tickerbox device list [flags]
```

### Examples

```
  tickerbox device list
```

### Options

```
  -h, --help   help for list
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

