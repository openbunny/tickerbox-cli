## tickerbox ntp

NTP / time sync

### Options

```
  -h, --help   help for ntp
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
* [tickerbox ntp set](tickerbox_ntp_set.md)	 - Change NTP settings
* [tickerbox ntp settings](tickerbox_ntp_settings.md)	 - Show NTP settings
* [tickerbox ntp status](tickerbox_ntp_status.md)	 - Show NTP sync status

