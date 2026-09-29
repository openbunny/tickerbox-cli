## tickerbox config export

Capture every device config section as a JSON snapshot

```
tickerbox config export [flags]
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
      --host string        TickerBox base URL (default http://tickerbox.local, or $TICKERBOX_HOST)
  -j, --json               emit raw device JSON instead of a decoded table
      --retry int          retry an idempotent request this many times on a transient failure (default 2)
      --timeout duration   per-request timeout (default 10s)
```

### SEE ALSO

* [tickerbox config](tickerbox_config.md)	 - Device configuration snapshot: export, import, diff

