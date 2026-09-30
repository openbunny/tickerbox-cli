## tickerbox config diff

Show per-section field differences between the device and a saved snapshot

### Synopsis

Compares every section. A differing wifi/ap secret field is shown masked as ******** by default; --show-secrets compares and shows their real values instead.

```
tickerbox config diff <file> [flags]
```

### Examples

```
  tickerbox config diff backup.json
```

### Options

```
  -h, --help           help for diff
      --show-secrets   compare wifi/ap passwords and keys instead of redacting them
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

* [tickerbox config](tickerbox_config.md)	 - Device configuration snapshot: export, import, diff

