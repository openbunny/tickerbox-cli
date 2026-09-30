## tickerbox wifi settings

Show Wi-Fi station settings

### Synopsis

Password is masked as ******** unless --show-secrets is given.

```
tickerbox wifi settings [flags]
```

### Examples

```
  tickerbox wifi settings --show-secrets
```

### Options

```
  -h, --help           help for settings
      --show-secrets   reveal the Wi-Fi password instead of masking it
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

* [tickerbox wifi](tickerbox_wifi.md)	 - Wi-Fi station configuration

