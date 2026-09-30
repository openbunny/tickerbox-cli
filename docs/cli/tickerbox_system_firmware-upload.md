## tickerbox system firmware-upload

Upload and flash new firmware

### Synopsis

Flashes FILE, which must end in .bin, replacing the running firmware. Cannot be undone. Prompts for confirmation unless --yes.

```
tickerbox system firmware-upload <file> [flags]
```

### Examples

```
  tickerbox system firmware-upload firmware.bin --yes
```

### Options

```
  -h, --help   help for firmware-upload
      --yes    skip confirmation
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

