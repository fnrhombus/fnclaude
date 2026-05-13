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
When `--name`/`-n` isn't already set, fnclaude generates a 1–3-word session label from the prompt
via Haiku (see [Auto-name from prompt](#auto-name-from-prompt) below).

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

`auto.tmux = "worktree"` injects `--tmux` only when you're explicitly creating a new worktree via
`-w` (which claude requires for `--tmux` anyway). fnclaude never spawns worktrees on its own —
that's always a user-initiated action.


### Auto-name from prompt

When you pass a prompt after `--` (and aren't resuming an existing session), fnclaude generates a
short hyphenated session label via Haiku and injects it as `--name`. The call has a 3-second
timeout; on timeout, missing API key, or any error, it falls back to a heuristic that strips
stop-words and takes the first three meaningful tokens.

```sh
fnclaude . -- "refactor the auth module"
# → --name refactor-auth-module
```

Skipped for `-p`/`--print`, `-r`/`--resume`, `-c`/`--continue`, and `--from-pr` — those don't
create new named sessions. Requires `ANTHROPIC_API_KEY`; suppress the missing-key warning with
`FNCLAUDE_QUIET_MISSING_API_KEY=1` (or the config equivalent) if you'd rather just rely on the
heuristic.


### Cross-cwd `--resume`

`claude --resume` normally exits with a "this conversation is from a different directory" message
when you pick a session from elsewhere via the picker. fnclaude scans the last 4 KB of claude's
output, catches that message after exit, and transparently `syscall.Exec`s a fresh fnclaude in the
destination directory. The picker just _works_ across all your projects — no flicker, no manual
`cd`.

Linux and macOS only; on Windows fnclaude falls back to a plain exec (no PTY, no detection).


### Worktree intercept

`fnclaude -w <name>` looks up `<name>` against the existing git worktrees of the project repo.
If it matches, fnclaude swaps its cwd to that worktree and drops the `-w` flag — no new wt is
created, no duplicate. If it doesn't match, the flag passes through and the name doubles as the
session `--name`.

```sh
fnclaude -w feature-branch    # cds to the feature-branch worktree if it exists
fnclaude -w new-thing         # passes -w new-thing through; sets --name new-thing
```

Shell completion (zsh, bash, fish) suggests existing worktree names for `-w` / `--worktree`.


### Shell completion

Completions for zsh, bash, and fish ship in the `completions/` directory. All three include smart
`-w` / `--worktree` completion that calls `git worktree list` to enumerate existing worktree
basenames.

- **zsh** — copy or symlink `completions/_fnclaude` to a directory in `$fpath`, then run `compinit`.
- **bash** — `source` `completions/fnclaude.bash` from your `.bashrc`.
- **fish** — copy `completions/fnclaude.fish` to `~/.config/fish/completions/`.


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

Linux is the daily-driver target. macOS and Windows binaries ship in every release; on Windows,
cross-cwd resume isn't available yet (it needs a PTY shim that Windows console makes painful).
Everything else works identically.


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

Everything else — multi-dir injection, short flags, MCP config auto-wiring — works the same.
Just remove the `cclaude` function from your shell aliases and call `fnclaude` (or the `fnc`
shortcut once the shell completion is loaded) directly.


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
| `-v` | `--version` | Print fnclaude's version and exit (shadows `claude`'s `-v`; use `claude --version` for that) |

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
model = "claude-haiku-4-5"   # model for auto-generated session names
timeout = "3s"               # timeout for the name-generation API call
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

`ANTHROPIC_API_KEY` is read (standard) for the auto-name LLM call.

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

Controls auto-injection of `--tmux`. claude rejects `--tmux` unless `--worktree`
is also set, so only two modes are valid:

- `"never"` (default) — no-op.
- `"worktree"` — inject `--tmux` when the user is creating a *new* worktree
  (`-w <new-name>` that didn't match an existing one). `--worktree` is
  necessarily present, so claude's constraint is satisfied. fnclaude never
  auto-creates worktrees on its own — they're always user-initiated.

Unknown values (including the deprecated `"always"`) are normalized to
`"never"` with a stderr warning that surfaces after claude exits.

Suppress for a single invocation with `--no-tmux`.

### Auto `--ide`

`auto.ide = "always"` appends `--ide` to every launch. No per-invocation
opt-out flag; unset the env var or config key to disable.

## Roadmap

- **Windows cross-cwd resume** — currently Linux/macOS only; needs a Windows
  console PTY shim.
- **winget submission** — manifest scaffolding is in `packaging/winget/`;
  pending a Windows verification pass.
- **mise / aqua install path** — not yet packaged for either.

## Contributing

After cloning, install the pre-commit hook so unformatted Go sources can't sneak through:

```sh
git config core.hooksPath .githooks
```

Or, if you use [mise](https://mise.jdx.dev/):

```sh
mise run setup
```

Both do the same thing — they point git at the tracked `.githooks/` directory. The hook itself (`.githooks/pre-commit`) runs `gofmt -l ./src` and blocks the commit if anything is unformatted. Fix with `gofmt -w ./src` and re-stage.

## License

MIT — see [LICENSE](LICENSE).
