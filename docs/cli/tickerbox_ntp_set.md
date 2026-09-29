## tickerbox ntp set

Change NTP settings

```
tickerbox ntp set [flags]
```

### Options

```
      --disabled        disable NTP sync
      --enabled         enable NTP sync
  -h, --help            help for set
      --server string   NTP server hostname or IP
      --tz string       timezone label
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

* [tickerbox ntp](tickerbox_ntp.md)	 - NTP / time sync

