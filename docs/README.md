# Documentation index

| Document                                                     | Covers                                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| [cli/tickerbox.md](cli/tickerbox.md)                         | CLI reference: every command and flag, generated from the cobra command tree.                    |
| [config-schema.md](config-schema.md)                         | On-disk device config (`config.toml`) and profile store: location, permissions, field reference. |
| [schema/snapshot.schema.json](schema/snapshot.schema.json)   | JSON Schema for a saved profile snapshot, referenced from `config-schema.md`.                    |
| [../internal/tz/PROVENANCE.md](../internal/tz/PROVENANCE.md) | Source and regeneration record for the bundled IANA timezone data.                               |

## Generating the CLI reference

`docs/gen/main.go` walks the cobra command tree and writes one Markdown file
per command into `cli/`. Run it with:

```console
$ just cli-docs
```

> [!NOTE]
> `just cli-docs` overwrites every file under `cli/`. Run it after any change
> to a command's flags, short/long description, or subcommand tree, and
> commit the regenerated files alongside that change.

Man pages generate from the same command tree with `just man`, into
`dist/man/` (not committed).
