# Cutting a release

Releases are driven by [release-please](https://github.com/googleapis/release-please): every push to `main` is parsed for [conventional commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `feat!:`, etc.) and an open "Release v0.X.Y" PR is kept up to date with the next version's CHANGELOG. Merging that PR cuts the release.

## 1. Author commits in conventional form

| Type | Effect | Example |
|---|---|---|
| `feat:` | minor version bump | `feat: add --no-tmux opt-out` |
| `fix:` | patch version bump | `fix: handle bare -A at end of argv` |
| `feat!:` / `BREAKING CHANGE:` in body | major version bump | `feat!: rename --also to --add` |
| `docs:`, `refactor:` | no bump (shown in changelog) | `docs: clarify magic positional rules` |
| `build:`, `ci:`, `chore:`, `test:` | no bump, hidden from changelog | `chore: bump go.mod to 1.25` |

Bodies and footers are free-form. Multi-line is fine.

## 2. Merge the Release PR

`release-please` keeps a `chore(main): release v0.X.Y` PR open at all times when there are releasable commits on `main`. Review the proposed version + CHANGELOG, merge it. release-please:

- Tags `vX.Y.Z` on `main`
- Creates a GitHub Release with the generated CHANGELOG section
- Updates `.release-please-manifest.json`

## 3. goreleaser fires automatically

The `release.yml` workflow listens for any `v*` tag — when release-please pushes the tag, goreleaser builds all targets and uploads:

- `fnclaude_Linux_x86_64.tar.gz`
- `fnclaude_Linux_arm64.tar.gz`
- `fnclaude_Darwin_x86_64.tar.gz`
- `fnclaude_Darwin_arm64.tar.gz`
- `fnclaude_Windows_x86_64.zip`
- `checksums.txt`

## 4. AUR package (`fnclaude-bin`)

After the GitHub Release is up:

1. Grab the sha256 for `fnclaude_Linux_x86_64.tar.gz` from `checksums.txt`.
2. In `packaging/aur/PKGBUILD`, set `pkgver=0.1.0` and replace `sha256sums=('SKIP')` with the real hash.
3. Regenerate `.SRCINFO`:
   ```sh
   makepkg --printsrcinfo > packaging/aur/.SRCINFO
   ```
4. Push both files to the `fnclaude-bin` AUR git repo.

## 5. winget

After the GitHub Release is up:

1. In `packaging/winget/`, bump `PackageVersion` in all three manifests.
2. In `installer.yaml`, update `InstallerUrl` and `InstallerSha256` (sha256 of the `.zip`, uppercase hex).
3. Fork `microsoft/winget-pkgs`, copy the three manifests into:
   ```
   manifests/f/fnrhombus/fnclaude/<version>/
   ```
   following the `fnrhombus.fnclaude` identifier path.
4. Open a PR against `microsoft/winget-pkgs`.
