## tickerbox completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(tickerbox completion zsh)

To load completions for every new session, execute once:

#### Linux:

	tickerbox completion zsh > "${fpath[1]}/_tickerbox"

#### macOS:

	tickerbox completion zsh > $(brew --prefix)/share/zsh/site-functions/_tickerbox

You will need to start a new shell for this setup to take effect.


```
tickerbox completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
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

