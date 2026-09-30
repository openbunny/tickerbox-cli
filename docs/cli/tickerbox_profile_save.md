## tickerbox profile save

Capture the device's current config as a named profile

### Synopsis

Captures tickers, display, clock, and ntp by default. --include selects specific sections by name. --all also captures wifi and ap, including their passwords. Prompts for confirmation if name is already saved, unless --yes.

```
tickerbox profile save <name> [flags]
```

### Examples

```
  tickerbox profile save home
  tickerbox profile save full --all
```

### Options

```
      --all                  also capture wifi and ap, including their secrets
  -D, --description string   optional human-readable description to store with the profile
  -h, --help                 help for save
      --include string       comma-separated sections to capture (default tickers,display,clock,ntp)
  -y, --yes                  skip confirmation
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

