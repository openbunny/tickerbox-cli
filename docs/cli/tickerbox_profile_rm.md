## tickerbox profile rm

Delete a saved profile

```
tickerbox profile rm <name> [flags]
```

### Options

```
  -h, --help   help for rm
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

* [tickerbox profile](tickerbox_profile.md)	 - Named, device-agnostic device configurations

