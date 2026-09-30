## tickerbox profile diff

Compare a saved profile against the device's current config

### Synopsis

Compares only the sections the profile contains. --show-secrets reveals wifi/ap password fields instead of masking them.

```
tickerbox profile diff <name> [flags]
```

### Examples

```
  tickerbox profile diff home
```

### Options

```
  -h, --help           help for diff
      --show-secrets   reveal wifi/ap password fields instead of masking them
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

* [tickerbox profile](tickerbox_profile.md)	 - Named, device-agnostic device configurations

