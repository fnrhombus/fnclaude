# Changelog

All notable changes to fnclaude are documented here. From v0.1.0 onward this file is maintained automatically by [release-please](https://github.com/googleapis/release-please) based on [conventional commits](https://www.conventionalcommits.org/) on `main`.

## [1.4.0](https://github.com/fnrhombus/fnclaude/compare/v1.3.0...v1.4.0) (2026-05-14)


### Features

* add fork subcommand for --resume + --fork-session ([#19](https://github.com/fnrhombus/fnclaude/issues/19)) ([941ce6f](https://github.com/fnrhombus/fnclaude/commit/941ce6f9bb85495ce4f535547258e5c9b76cd0fa))

## [1.3.0](https://github.com/fnrhombus/fnclaude/compare/v1.2.0...v1.3.0) (2026-05-14)


### Features

* allow / in worktree/branch names, block path escape ([#17](https://github.com/fnrhombus/fnclaude/issues/17)) ([aa44de8](https://github.com/fnrhombus/fnclaude/commit/aa44de8249487b9872b370b71daa49520561ebbb))

## [1.2.0](https://github.com/fnrhombus/fnclaude/compare/v1.1.0...v1.2.0) (2026-05-13)


### Features

* **config:** inject configurable env vars into claude's process env ([#11](https://github.com/fnrhombus/fnclaude/issues/11)) ([f0c0e3b](https://github.com/fnrhombus/fnclaude/commit/f0c0e3bddf16e4136a47d73de5b4d715be614ef3))


### Bug Fixes

* **ci:** pass --repo to gh in release-please workflow ([#13](https://github.com/fnrhombus/fnclaude/issues/13)) ([287e3aa](https://github.com/fnrhombus/fnclaude/commit/287e3aa295b927efd4af8e7ab0f492ed15c677d0))
* **ci:** use AUTOMERGE_PAT so auto-merges trigger downstream workflows ([74c9890](https://github.com/fnrhombus/fnclaude/commit/74c98900a1a274969523684d85647c797315c5fd))
* **ci:** use AUTOMERGE_PAT so auto-merges trigger downstream workflows ([#14](https://github.com/fnrhombus/fnclaude/issues/14)) ([74c9890](https://github.com/fnrhombus/fnclaude/commit/74c98900a1a274969523684d85647c797315c5fd))

## [1.1.0](https://github.com/fnrhombus/fnclaude/compare/v1.0.0...v1.1.0) (2026-05-13)


### Features

* add resume/continue subcommands (res/con shorthands) ([#6](https://github.com/fnrhombus/fnclaude/issues/6)) ([d260ed7](https://github.com/fnrhombus/fnclaude/commit/d260ed75ec0b0994119876c4b0c108b9fd6ea157))
* sanitize user-supplied --name to path-safe chars ([#5](https://github.com/fnrhombus/fnclaude/issues/5)) ([3d59d4c](https://github.com/fnrhombus/fnclaude/commit/3d59d4cd8e433bd3309a362bcc47f691b7210323))

## [1.0.0](https://github.com/fnrhombus/fnclaude/compare/v0.1.0...v1.0.0) (2026-05-13)


### ⚠ BREAKING CHANGES

* auto.tmux is now "never" | "worktree" only.

### Features

* **aur:** symlink /usr/bin/fnc -&gt; fnclaude in fnclaude-bin package ([422bc97](https://github.com/fnrhombus/fnclaude/commit/422bc975516d30f3cd94bc31951ffbb494511343))
* drop auto.tmux="always"; defer non-fatal warnings until after claude exits ([fcb96a9](https://github.com/fnrhombus/fnclaude/commit/fcb96a9c5455ab254615e06473bd0624b286703f))
* **worktree-match:** match by basename, branch, or worktree-stripped branch ([0e4c436](https://github.com/fnrhombus/fnclaude/commit/0e4c436bb7ff7925e17251f10882c8e693b10f83))


### Bug Fixes

* **auto-tmux:** inject --worktree alongside --tmux to satisfy claude's constraint ([e7f477f](https://github.com/fnrhombus/fnclaude/commit/e7f477fe87722af05fc761dd4a300331343555e6))
* **resume:** anchor cross-cwd regex on plain-text portions of TUI output ([#3](https://github.com/fnrhombus/fnclaude/issues/3)) ([c01f245](https://github.com/fnrhombus/fnclaude/commit/c01f245caaa9189a18f0eddceadb123a78e0eb8f))


### Documentation

* promote shipped features (auto-name, cross-cwd resume, worktree intercept, completion) into Features; trim Roadmap ([d196d4e](https://github.com/fnrhombus/fnclaude/commit/d196d4e1fd4f624ce44dcf3761c9b6e6559f533f))
* rewrite README as pitch-first with reference at the bottom ([d3081fd](https://github.com/fnrhombus/fnclaude/commit/d3081fdf002cd4368723e4249ff7ada29e8f546c))


### Refactoring

* organize Go source files under src/ ([8c92524](https://github.com/fnrhombus/fnclaude/commit/8c9252499a7f63cdb340aa0570dc403060bdd77d))
* **worktree-match:** branch-first, then stripped-branch, then basename ([376aaad](https://github.com/fnrhombus/fnclaude/commit/376aaad6ca66d5fbba056626880caeb7751b8945))

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
