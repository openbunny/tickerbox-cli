## tickerbox device

Manage configured devices

### Synopsis

Manages named devices this CLI can target: add one with `device add <name> <host>`, pick which is used by default with `device use <name>`, or point a single call at any device with --host or --device without changing the default. See `tickerbox profile` for named, device-agnostic setting bundles, and `tickerbox config` for a raw export/import/diff of one device's live snapshot.

### Options

```
  -h, --help   help for device
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
* [tickerbox device add](tickerbox_device_add.md)	 - Add or replace a configured device
* [tickerbox device current](tickerbox_device_current.md)	 - Show the device the CLI currently targets
* [tickerbox device discover](tickerbox_device_discover.md)	 - Find TickerBoxes on the local network
* [tickerbox device list](tickerbox_device_list.md)	 - List configured devices
* [tickerbox device ping](tickerbox_device_ping.md)	 - Check reachability of one or all configured devices
* [tickerbox device rm](tickerbox_device_rm.md)	 - Remove a configured device
* [tickerbox device use](tickerbox_device_use.md)	 - Set the default device

