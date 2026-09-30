## tickerbox config export

Capture every device config section as a JSON snapshot

### Synopsis

Captures every config section, including wifi and ap. --show-secrets includes their passwords; otherwise they're omitted from the file.

```
tickerbox config export [flags]
```

### Examples

```
  tickerbox config export --file backup.json
```

### Options

```
      --file string    file to write the snapshot to; defaults to stdout
  -h, --help           help for export
      --show-secrets   include wifi/ap passwords and keys in the export
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

* [tickerbox config](tickerbox_config.md)	 - Device configuration snapshot: export, import, diff

