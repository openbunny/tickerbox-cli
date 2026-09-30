## tickerbox wifi set

Update Wi-Fi station settings

### Synopsis

Updates only the fields given as flags; every other field keeps its current device value. --password prompts on a terminal, reads from --password-stdin, or is passed directly; setting a password over a plain http:// host prints a warning unless --yes. --static-ip and --no-static-ip are mutually exclusive.

```
tickerbox wifi set [flags]
```

### Examples

```
  tickerbox wifi set --ssid HomeNet --password-stdin < wifi.secret
  tickerbox wifi set --static-ip --local-ip 192.168.1.50 --gateway-ip 192.168.1.1 --subnet-mask 255.255.255.0
```

### Options

```
      --dns1 string          primary DNS server
      --dns2 string          secondary DNS server
      --gateway-ip string    static gateway IP address
  -h, --help                 help for set
      --hostname string      device hostname
      --local-ip string      static local IP address
      --no-static-ip         use DHCP instead of a static IP
      --password string      Wi-Fi network password
      --password-stdin       read the Wi-Fi network password from stdin
      --ssid string          Wi-Fi network name (max 32 characters)
      --static-ip            use a static IP instead of DHCP
      --subnet-mask string   static subnet mask
  -y, --yes                  skip the plaintext-HTTP password warning
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

* [tickerbox wifi](tickerbox_wifi.md)	 - Wi-Fi station configuration

