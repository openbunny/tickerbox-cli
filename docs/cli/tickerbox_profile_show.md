## tickerbox profile show

Show a saved profile

### Synopsis

Password is masked as ******** unless --show-secrets is given.

```
tickerbox profile show <name> [flags]
```

### Examples

```
  tickerbox profile show home
```

### Options

```
  -h, --help           help for show
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

