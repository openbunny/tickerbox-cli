## tickerbox system factory-reset

Erase all device settings and restore factory defaults

### Synopsis

Erases all device settings and cannot be undone. Prompts for confirmation unless --yes. --backup-first writes the current config to tickerbox-backup-<UTC timestamp>.json before wiping.

```
tickerbox system factory-reset [flags]
```

### Examples

```
  tickerbox system factory-reset --backup-first --yes
```

### Options

```
      --backup-first   export the device config to a timestamped file before wiping it
  -h, --help           help for factory-reset
      --yes            skip confirmation
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

* [tickerbox system](tickerbox_system.md)	 - System, status, maintenance

