# fnclaude

`claude`, with the rough edges filed off.

```sh
fnclaude opus max ~/src/myproject -- "refactor the auth module"
```

That's it. No `--model claude-opus-4-5`, no `--effort max`, no `--print` gymnastics. fnclaude is a
thin Go binary sitting in front of `claude` that translates short, readable invocations into the
full-form flags `claude` expects. Magic positional words for model and effort, capital-letter short
flags for everything claude makes you spell out, and a config file for the auto-features you want
on every launch. Static binary, ~2.6 MB, no runtime.


## Features

### Magic positional words

The first two positional slots can be a model alias and an effort level. fnclaude intercepts them
before `claude` ever sees the args.

```sh
fnclaude opus max ~/src/proj          # --model claude-opus-4-5 --effort max
fnclaude sonnet ~/src/proj            # --model claude-sonnet-4-5
fnclaude haiku low ~/src/proj         # --model claude-haiku-4-5 --effort low
fnclaude ~/src/proj                   # no model flag — claude picks the default
```

Supported model aliases: `opus`, `sonnet`, `haiku`.
Supported effort levels: `low`, `medium`, `high`, `xhigh`, `max`.

A directory that happens to be named `opus`? Prefix it: `fnclaude ./opus`.


### Capital-letter short flags

`claude`'s long options are the right thing to pass and a chore to type. fnclaude maps each one to
a capital-letter short flag that collapses with standard POSIX rules.

```sh
fnclaude -BVC ~/src/proj     # --brief --verbose --chrome
fnclaude -T ~/src/proj       # --tmux
fnclaude -D ~/src/proj       # --dangerously-skip-permissions
```

Full mapping in the [reference section](#short-translations-fnclaude--claude) below.


### Prompt after `--`

Pass a prompt inline without `--print` or redirection — just drop a `--` and write the prompt.
Session naming from the prompt text is on the roadmap; for now the prompt is forwarded verbatim.

```sh
fnclaude sonnet src/ -- "add integration tests for the payments module"
```


### Multi-directory MCP injection

Need claude to see a second project's MCP config and settings? Pass it as an extra positional or
with `-A`/`--also`. fnclaude injects `--add-dir`, `--mcp-config`, and `--settings` for each extra
dir automatically.

```sh
fnclaude src/ -A tools/ -A shared/
# or equivalently:
fnclaude src/ tools/ shared/
```

Each extra dir gets `--mcp-config <dir>/.mcp.json` (if the file exists) and
`--settings <dir>/.claude/settings.json` (if it exists). The primary dir is launched in; extra dirs
are attached.


### Auto-features you configure once

Tired of typing `--tmux` every launch? Set it once in config and forget it.

```toml
# ~/.config/fnclaude/config.toml
[auto]
tmux = "always"                      # inject --tmux on every launch
ide = "always"                       # inject --ide on every launch
dangerously_skip_permissions = true  # inject -D on every launch
```

All three are off by default. Per-invocation overrides (`--no-tmux`, `--no-permissions`) let you
escape for a single run without touching config.


### Install

**AUR** (Arch Linux):

```sh
yay -S --rmdeps fnclaude-bin
```

**Go install** (any platform with Go):

```sh
go install github.com/fnrhombus/fnclaude@latest
```

**GitHub Releases** — grab the binary for your platform from the
[releases page](https://github.com/fnrhombus/fnclaude/releases).

winget and mise packages are planned but not yet available.

Linux is the daily-driver target. macOS and Windows binaries ship in every release; Windows skips
PTY-dependent features (cross-cwd resume, worktree intercept) because Windows console makes those
painful to implement correctly.


## Quickstart

```sh
# Launch in the current directory
fnclaude .

# Launch with a specific model in a specific project
fnclaude sonnet ~/src/myproject

# High-effort opus session
fnclaude opus high ~/src/myproject

# Attach a shared tools directory
fnclaude ~/src/myproject -A ~/src/shared-tools

# Pass a prompt inline
fnclaude . -- "refactor the auth module"

# Skip tmux auto-attach for this run when you have auto.tmux = always set
fnclaude . --no-tmux

# Collapse multiple short flags
fnclaude -BVC .
```


## Migration from cclaude

fnclaude is the Go rewrite of the `cclaude` zsh function. The interface is intentionally close;
most invocations translate directly. Two things changed:

1. **`--dangerously-skip-permissions` is opt-in.** `cclaude` passed it unconditionally; fnclaude
   does not. Enable it globally with `auto.dangerously_skip_permissions = true` in config, or
   per-invocation with `-D`.
2. **No `-i`/`--init` flag.** Dropped; use `claude`'s native init flow if you need it.

Everything else — multi-dir injection, short flags, MCP config auto-wiring — works the same. Point
your existing `cclaude` alias at `fnclaude` and it'll behave.


## Support

If fnclaude saves you time, you can support its development via
[GitHub Sponsors](https://github.com/sponsors/fnrhombus) or
[Buy Me a Coffee](https://buymeacoffee.com/fnrhombus).


---

# Reference

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

## Flag reference

### fnclaude-owned flags

| Flag | Long | Description |
|---|---|---|
| | `--no-tmux` | Suppress auto-`--tmux` for this invocation |
| | `--no-permissions` | Suppress auto-`--dangerously-skip-permissions` for this invocation |
| `-A <dir>` | `--also <dir>` | Add an extra dir (repeatable; supports `=` syntax) |
| `-h` | `--help` | Print the fnclaude flag reference and exit |

### Short translations (fnclaude → claude)

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

## Roadmap

The following features are not yet shipped:

- **Auto `--name` from prompt** — when `--` is followed by a prompt, call Haiku
  4.5 to generate a 1-3 word session label, with a heuristic fallback on
  timeout.
- **Cross-cwd `--resume`** — transparent re-launch when `claude` exits with the
  "different directory" message.
- **Worktree intercept** — `-w existing-wt-name` cds to that worktree;
  `-w new-wt-name` has `claude` create the worktree and fnclaude sets `--name`
  automatically.
- **Shell completion** — zsh, bash, and fish.

## License

MIT — see [LICENSE](LICENSE).
