# Working on fnclaude

## Branch and worktree workflow — HARD RULE

**No direct commits to `main`.** All changes land via PR from a feature branch + worktree. `main` is protected; even the maintainer goes through the PR flow.

For every change, create a worktree:

```sh
git worktree add ../fnclaude+<feature-name> -b <feature-name>
cd ../fnclaude+<feature-name>
# … work, commit, push the branch …
gh pr create --fill
```

When the PR merges, **clean up immediately** in the same shell session:

```sh
cd <main-worktree>
git pull --ff-only
git worktree remove ../fnclaude+<feature-name>
git branch -d <feature-name>
git push origin :<feature-name>   # only if you pushed the branch
```

Dangling feature branches or stray worktrees are smells:

- **Worktree still around after merge** → either the work isn't actually done (finish it) or the work was merged and you forgot to clean up. Don't accumulate worktrees "just in case" — start a new one when you need it.
- **Branch still on origin after PR merge** → delete via `gh pr` (GitHub's web-UI checkbox) or `git push origin :<branch>`.

Use `mise run worktree-new <name>` once that task exists; until then the snippet above is the canonical recipe.

## Release flow

This repo uses [release-please](https://github.com/googleapis/release-please).

- Every push to `main` (necessarily via PR merge — see above) triggers the `Release Please` workflow.
- release-please keeps an open `chore(main): release vX.Y.Z` PR continuously up to date with the proposed next version and an auto-generated `CHANGELOG.md` derived from conventional-commit messages.
- That PR **auto-merges** once `test` is green (configured in `.github/workflows/release-please.yml`). You don't manually merge it.
- The release-please merge tags `vX.Y.Z`. The `release.yml` workflow fires on the tag, goreleaser cross-builds the binaries, and the `publish-aur` job pushes the version bump to AUR.

Effectively: every PR merge to `main` ships a release. There's no "save up a few commits then release" intermediate state — `main` is always shipped.

### Version bump rules (conventional commits)

| commit type | bump | shown in CHANGELOG |
|---|---|---|
| `feat:` | minor (0.X.0) | yes |
| `fix:` | patch (0.0.X) | yes |
| `feat!:` or `BREAKING CHANGE:` in body | major (X.0.0) | yes |
| `docs:`, `refactor:`, `perf:`, `revert:` | none | yes |
| `chore:`, `ci:`, `build:`, `test:` | none | hidden |

## Commit conventions

- **Format:** `<type>(<scope>): <subject>` per [conventional commits](https://www.conventionalcommits.org/). Subject under ~70 chars; body explains the *why*.
- **Author:** `fnrhombus`, email `2511516+fnrhombus@users.noreply.github.com` (the GitHub noreply form — never the underlying gmail).
- **No `Co-Authored-By: Claude` trailer.** No AI attribution anywhere visible — commit messages, PR bodies, issue replies. See `~/.claude/CLAUDE.md` for the longer version of this rule.
- **No `--no-verify`** to bypass pre-commit hooks. If a hook fails, investigate and fix the underlying issue (usually `chezmoi re-add <file>` for unrelated drift).

## Layout

- `src/` — Go source for the binary
- `bin/` — build output (`mise run build`); gitignored
- `completions/` — zsh + bash + fish completion scripts
- `packaging/` — AUR PKGBUILD + winget manifests + goreleaser config
- `.github/workflows/` — `test`, `release-please`, `release`
- `mise.toml` — Go pin + build/test/install-dev tasks
- `.goreleaser.yaml` + `release-please-config.json` + `.release-please-manifest.json` — release machinery
