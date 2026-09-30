## tickerbox device rm

Remove a configured device

### Synopsis

Removes a configured device. Prompts for confirmation unless --yes.

```
tickerbox device rm <name> [flags]
```

### Examples

```
  tickerbox device rm desk --yes
```

### Options

```
  -h, --help   help for rm
  -y, --yes    skip confirmation
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

