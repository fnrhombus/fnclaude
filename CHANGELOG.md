# Changelog

All notable changes to fnclaude are documented here. From v0.1.0 onward this file is maintained automatically by [release-please](https://github.com/googleapis/release-please) based on [conventional commits](https://www.conventionalcommits.org/) on `main`.

## v0.1.0 (2026-05-11)

Initial release. fnclaude is a Go rewrite of the long-running `cclaude` zsh function, promoted to its own cross-platform CLI with quality-of-life features layered on top.

### Features

- **Parity with cclaude.** Leading positional paths become claude's cwd + `--add-dir` extras, with `.mcp.json` and `.claude/settings.json` auto-injected from each extra dir that has them.
- **Magic positional words.** Before any path, position 1 may be a model alias (`opus` | `sonnet` | `haiku`) and position 2 (only when position 1 was a model) may be an effort level (`low` | `medium` | `high` | `xhigh` | `max`). Example: `fnclaude opus max ~/src/proj`.
- **Capital-letter short flags.** fnclaude reserves the uppercase short-flag namespace and translates each to claude's long form: `-A → --also`, `-B → --brief`, `-C → --chrome`, `-D → --dangerously-skip-permissions`, `-F → --fork-session`, `-G → --agent`, `-I → --ide`, `-M → --permission-mode`, `-P → --from-pr`, `-R → --remote-control`, `-T → --tmux`, `-V → --verbose`, `-W → --allowedTools`. POSIX collapsing supported (`-BVC` = `-B -V -C`).
- **fnclaude-owned flags:** `-A/--also <dir>` (extra-dir, repeatable), `--no-tmux` and `--no-permissions` (per-invocation suppressors), `-h/--help`, `-v/--version`.
- **Config file** at `$XDG_CONFIG_HOME/fnclaude/config.toml` (TOML) with env-var equivalents. Precedence: CLI > env > config > default.
- **Auto-features (off by default; enable via config / env):**
  - `auto.tmux = "always" | "worktree" | "never"` — auto-injects `--tmux`.
  - `auto.ide = "always" | "never"` — auto-injects `--ide`.
  - `auto.dangerously_skip_permissions = true | false` — auto-injects `--dangerously-skip-permissions`.
- **Auto-`--name` from prompt.** When `--` is followed by a prompt and `--name`/`-n` isn't already set, fnclaude generates a 1–3-word session label via Claude Haiku 4.5, falling back to a heuristic on timeout or missing API key. Configurable model, timeout, and warning-suppression via `[name]` config section.
- **Transparent cross-cwd `--resume`.** When claude's resume picker exits with the "different directory" message, fnclaude detects it via a PTY-buffered scan and silently relaunches itself in the destination cwd with `--resume <uuid>` — preserves the user's other flags, magic words, etc.
- **Worktree intercept.** `-w <name>` matching an existing worktree of the project repo swaps fnclaude's cwd to that worktree (drops the `-w`). Non-matching names pass through and the wt name doubles as the session `--name`.
- **Agent-worktree-pitfall warning** auto-appended to claude's system prompt (skipped for `-p`/`--print`). Warns LLMs spawning Agents with `isolation: "worktree"` to not name the main repo's path in the spawn prompt — a real foot-gun where agents `cd` to a named path and silently bypass their isolated worktree.

### Tooling

- **Shell completion** scripts for zsh, bash, and fish — including smart `-w`/`--worktree` completion from `git worktree list`.
- **Release pipeline:** [release-please](https://github.com/googleapis/release-please) on every push to `main` for conventional-commit-driven semver + changelog; [goreleaser](https://goreleaser.com/) cross-builds linux/darwin/windows binaries (amd64+arm64) on each `v*` tag.
- **Distribution scaffolds** for AUR (`fnclaude-bin`) and winget (`fnrhombus.fnclaude`).

### Migration from cclaude

cclaude's zsh function in `dot_zsh_aliases` has been replaced with `alias cclaude=fnclaude`, preserving muscle memory during the transition. Major behavioral changes vs cclaude:

- `-m/--no-mcp` and `-s/--no-settings` removed — use the raw `--add-dir` claude flag (no auto-injection) to express the same intent more cleanly.
- `-i/--init <prompt>` removed — use `--` followed by the prompt (e.g. `fnclaude src/ -- "do the thing"`); auto-`--name` fires from that prompt.
- `--dangerously-skip-permissions` is no longer added unconditionally — enable via config (`auto.dangerously_skip_permissions = true`) or pass `-D` per invocation.
