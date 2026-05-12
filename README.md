# fnclaude

A wrapper around the `claude` CLI that adds quality-of-life argument shortcuts,
magic positional words for common options, multi-directory MCP config injection,
auto-features controlled by a config file, and a clean single-binary install.
Linux-only for now (v1); Mac and Windows are on the roadmap. Static binary,
~2.6 MB, no runtime dependencies.

This is a Go rewrite of the `cclaude` zsh function.

## Install

```sh
go install github.com/fnrhombus/fnclaude@latest
```

AUR, winget, and mise packages are planned but not yet available.

## Quick examples

```sh
# Launch in the current directory
fnclaude .

# Launch in ~/src/myproject
fnclaude ~/src/myproject

# Use the sonnet model
fnclaude sonnet ~/src/myproject

# Use haiku at low effort
fnclaude haiku low ~/src/myproject

# Attach a second directory (adds --add-dir and injects any MCP/settings it finds)
fnclaude ~/src/myproject --also ~/src/shared-tools

# Same with the short flag
fnclaude ~/src/myproject -A ~/src/shared-tools

# Pass a prompt via --
fnclaude . -- "refactor the auth module"

# Skip tmux auto-attach for this run
fnclaude . --no-tmux

# Collapse short flags (brief + verbose + chrome)
fnclaude -BVC .

# Disambiguate a literal directory named "opus"
fnclaude ./opus
```

## Argument grammar

### Magic positional words

The first two positional arguments may be "magic" shorthands:

- **Position 1**: if the value is exactly `opus`, `sonnet`, or `haiku`, it is
  translated to `--model <alias>` and consumed. Otherwise it is treated as a
  path (magic stops).
- **Position 2**: only checked when position 1 was a model alias. If the value
  is exactly `low`, `medium`, `high`, `xhigh`, or `max`, it is translated to
  `--effort <level>` and consumed. Otherwise it is treated as a path (magic
  stops).
- **Position 3+**: never magic.

To pass a literal directory named `opus`, `sonnet`, etc., prefix it with `./`:

```sh
fnclaude ./opus
```

### Positional paths

After any magic slots are resolved:

- **First positional** = the directory to launch `claude` in. Falls back to
  `~/.claude/noop` when none is given.
- **Subsequent positionals** = "extra dirs". Each extra dir receives:
  - `--add-dir <dir>`
  - `--mcp-config <dir>/.mcp.json` (only if the file exists)
  - `--settings <dir>/.claude/settings.json` (only if the file exists)

The `-A`/`--also` flag is equivalent to a 2nd-or-later positional and may be
repeated:

```sh
fnclaude src/ -A tools/ -A shared/
```

Note: if `--setting-sources` appears anywhere in the passthrough args, fnclaude
suppresses its own `--settings` injection (the two flags are incompatible in
`claude`).

### Passing a prompt

Use `--` to separate fnclaude args from the prompt string:

```sh
fnclaude sonnet src/ -- "do the thing"
```

### Flag reference

#### fnclaude-owned flags

| Flag | Long | Description |
|---|---|---|
| | `--no-tmux` | Suppress auto-`--tmux` for this invocation |
| | `--no-permissions` | Suppress auto-`--dangerously-skip-permissions` for this invocation |
| `-A <dir>` | `--also <dir>` | Add an extra dir (repeatable; supports `=` syntax) |

#### Short translations (fnclaude → claude)

fnclaude adopts a capital-letter convention for its short flags. Each is
translated to the corresponding `claude` long form before the subprocess is
launched.

| Short | Long | Value |
|---|---|---|
| `-A` | `--also` | required (fnclaude-owned) |
| `-B` | `--brief` | none |
| `-C` | `--chrome` | none |
| `-D` | `--dangerously-skip-permissions` | none |
| `-F` | `--fork-session` | none |
| `-G` | `--agent` | required |
| `-I` | `--ide` | none |
| `-M` | `--permission-mode` | required |
| `-P` | `--from-pr` | optional |
| `-R` | `--remote-control` | optional |
| `-T` | `--tmux` | optional |
| `-V` | `--verbose` | none |
| `-W` | `--allowedTools` | required |

Short flags follow standard POSIX collapsing: `-BVC` expands to `-B -V -C`.
Only the last flag in a collapsed group may take a value.

All other flags are passed through to `claude` verbatim.

## Config file

Location: `$XDG_CONFIG_HOME/fnclaude/config.toml` (fallback
`~/.config/fnclaude/config.toml`). A missing file is not an error — all
defaults apply.

Precedence: **CLI flag > env var > config file > built-in default**

```toml
[name]
model = "claude-haiku-4-5"   # model for LLM-generated session names (roadmap)
timeout = "3s"               # timeout for name-generation calls (roadmap)
quiet_missing_api_key = false

[auto]
tmux = "never"                       # "never" | "worktree" | "always"
ide = "never"                        # "never" | "always"
dangerously_skip_permissions = false
```

### Env var mapping

| Config key | Env var |
|---|---|
| `name.model` | `FNCLAUDE_NAME_MODEL` |
| `name.timeout` | `FNCLAUDE_NAME_TIMEOUT` |
| `name.quiet_missing_api_key` | `FNCLAUDE_QUIET_MISSING_API_KEY` |
| `auto.tmux` | `FNCLAUDE_TMUX` |
| `auto.ide` | `FNCLAUDE_IDE` |
| `auto.dangerously_skip_permissions` | `FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS` |

`ANTHROPIC_API_KEY` is also read (standard) and will be used for
LLM-generated session name calls once that feature ships.

## Auto-features

### Auto `--dangerously-skip-permissions`

Off by default (unlike `cclaude`, which always passed it). Enable globally:

```toml
[auto]
dangerously_skip_permissions = true
```

or via `FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=true`, or per-invocation with
`-D`. Suppress a globally-enabled setting for one invocation with
`--no-permissions`.

### Auto `--tmux`

Controls whether fnclaude automatically appends `--tmux` to the claude
invocation:

- `"never"` (default) — no-op
- `"worktree"` — inject when `-w`/`--worktree` is present in the args
- `"always"` — inject on every launch

Suppress for a single invocation with `--no-tmux`.

### Auto `--ide`

`auto.ide = "always"` appends `--ide` to every launch. No per-invocation
opt-out flag; unset the env var or config key to disable.

## Migration from cclaude

`fnclaude` is a drop-in replacement for the `cclaude` zsh function with two
behavioural differences to be aware of:

1. **`--dangerously-skip-permissions` is opt-in.** `cclaude` passed it
   unconditionally; `fnclaude` does not unless configured or `-D` is used.
2. **No `-i`/`--init` flag.** That flag was dropped; use `claude`'s native
   init flow if needed.

Everything else — multi-dir injection, short flags, MCP config auto-wiring —
works the same way. If you previously aliased `cclaude`, point the alias at
`fnclaude` instead.

## Roadmap

The following features are not yet shipped:

- **Auto `--name` from prompt** — when `--` is followed by a prompt, call Haiku
  4.5 to generate a 1–3 word session label, with a heuristic fallback on
  timeout.
- **Cross-cwd `--resume`** — transparent re-launch when `claude` exits with the
  "different directory" message.
- **Worktree intercept** — `-w existing-wt-name` cds to that worktree;
  `-w new-wt-name` has `claude` create the worktree and fnclaude sets `--name`
  automatically.
- **Shell completion** — zsh, bash, and fish.
- **Mac + Windows support.**

## License

MIT — see [LICENSE](LICENSE).
