## tickerbox config import

Apply a saved config snapshot to the device

### Synopsis

Applies only the sections present in the file. Prompts for confirmation unless --yes.

```
tickerbox config import <file> [flags]
```

### Examples

```
  tickerbox config import backup.json --yes
```

### Options

```
  -h, --help   help for import
  -y, --yes    skip confirmation
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

