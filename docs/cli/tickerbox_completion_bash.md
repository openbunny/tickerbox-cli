## tickerbox completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(tickerbox completion bash)

To load completions for every new session, execute once:

#### Linux:

	tickerbox completion bash > /etc/bash_completion.d/tickerbox

#### macOS:

	tickerbox completion bash > $(brew --prefix)/etc/bash_completion.d/tickerbox

You will need to start a new shell for this setup to take effect.


```
tickerbox completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
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

* [tickerbox completion](tickerbox_completion.md)	 - Generate the autocompletion script for the specified shell

