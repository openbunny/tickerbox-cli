## tickerbox completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	tickerbox completion fish | source

To load completions for every new session, execute once:

	tickerbox completion fish > ~/.config/fish/completions/tickerbox.fish

You will need to start a new shell for this setup to take effect.


```
tickerbox completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
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

* [tickerbox completion](tickerbox_completion.md)	 - Generate the autocompletion script for the specified shell

