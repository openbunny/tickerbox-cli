## tickerbox device add

Add or replace a configured device

### Synopsis

Adds NAME as a device, or replaces it if the name already exists. HOST must include a scheme, e.g. http://tickerbox.local or http://192.168.1.42.

```
tickerbox device add NAME HOST [flags]
```

### Examples

```
  tickerbox device add desk http://tickerbox.local
```

### Options

```
  -h, --help   help for add
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

