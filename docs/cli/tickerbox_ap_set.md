## tickerbox ap set

Change access point settings

### Synopsis

Updates only the fields given as flags, merging into the current AP settings. --mode is one of always, disconnected, or never. --ssid is at most 32 characters, --password 8-64, --channel 1-14, --max-clients 1-9. --hidden/--no-hidden are mutually exclusive, as are --password and --password-stdin. Setting a password over plain http:// warns unless --yes.

```
tickerbox ap set [flags]
```

### Examples

```
  tickerbox ap set --mode disconnected --ssid TickerBox-Setup --password-stdin < ap.secret
  tickerbox ap set --channel 6 --max-clients 4
```

### Options

```
      --channel int          AP channel (1-14)
      --gateway-ip string    AP gateway IP
  -h, --help                 help for set
      --hidden               hide the AP SSID
      --local-ip string      AP local IP
      --max-clients int      max AP clients (1-9)
      --mode string          AP mode: always, disconnected, or never
      --no-hidden            broadcast the AP SSID
      --password string      AP password (8-64 chars)
      --password-stdin       read the AP password from stdin
      --ssid string          AP SSID (max 32 chars)
      --subnet-mask string   AP subnet mask
  -y, --yes                  skip the plaintext-HTTP password warning
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

* [tickerbox ap](tickerbox_ap.md)	 - Access point

