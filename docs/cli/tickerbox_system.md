## tickerbox system

System, status, maintenance

### Options

```
  -h, --help   help for system
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

* [tickerbox](tickerbox.md)	 - Control a TickerBox device over its REST API
* [tickerbox system factory-reset](tickerbox_system_factory-reset.md)	 - Erase all device settings and restore factory defaults
* [tickerbox system features](tickerbox_system_features.md)	 - Show enabled device features
* [tickerbox system firmware-upload](tickerbox_system_firmware-upload.md)	 - Upload and flash new firmware
* [tickerbox system info](tickerbox_system_info.md)	 - Show device system status
* [tickerbox system restart](tickerbox_system_restart.md)	 - Restart the device

