## tickerbox config

Device configuration snapshot: export, import, diff

### Synopsis

Raw export, import, and diff of one device's live config snapshot as a JSON file. Distinct from `tickerbox profile`, which stores named, reusable bundles rather than a one-off file, and unrelated to `tickerbox device`, which manages which device this CLI targets.

### Options

```
  -h, --help   help for config
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

* [tickerbox](tickerbox.md)	 - Control a TickerBox device over its REST API
* [tickerbox config diff](tickerbox_config_diff.md)	 - Show per-section field differences between the device and a saved snapshot
* [tickerbox config export](tickerbox_config_export.md)	 - Capture every device config section as a JSON snapshot
* [tickerbox config import](tickerbox_config_import.md)	 - Apply a saved config snapshot to the device

