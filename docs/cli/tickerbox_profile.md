## tickerbox profile

Named, device-agnostic device configurations

### Options

```
  -h, --help   help for profile
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
* [tickerbox profile apply](tickerbox_profile_apply.md)	 - Apply a saved profile to the device
* [tickerbox profile diff](tickerbox_profile_diff.md)	 - Compare a saved profile against the device's current config
* [tickerbox profile list](tickerbox_profile_list.md)	 - List saved profiles
* [tickerbox profile rm](tickerbox_profile_rm.md)	 - Delete a saved profile
* [tickerbox profile save](tickerbox_profile_save.md)	 - Capture the device's current config as a named profile
* [tickerbox profile show](tickerbox_profile_show.md)	 - Show a saved profile

