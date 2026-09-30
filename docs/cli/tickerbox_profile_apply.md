## tickerbox profile apply

Apply a saved profile to the device

### Synopsis

Applies only the sections present in the saved profile; sections it doesn't contain are left untouched on the device. Prompts for confirmation unless --yes.

```
tickerbox profile apply <name> [flags]
```

### Examples

```
  tickerbox profile apply home --yes
```

### Options

```
  -h, --help   help for apply
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

* [tickerbox profile](tickerbox_profile.md)	 - Named, device-agnostic device configurations

