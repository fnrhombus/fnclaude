# Cutting a release

## 1. Tag and push

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `release.yml` workflow fires on any `v*` tag. goreleaser builds all targets, creates the GitHub Release, and uploads:

- `fnclaude_Linux_x86_64.tar.gz`
- `fnclaude_Linux_arm64.tar.gz`
- `fnclaude_Darwin_x86_64.tar.gz`
- `fnclaude_Darwin_arm64.tar.gz`
- `fnclaude_Windows_x86_64.zip`
- `checksums.txt`

## 2. AUR package (`fnclaude-bin`)

After the GitHub Release is up:

1. Grab the sha256 for `fnclaude_Linux_x86_64.tar.gz` from `checksums.txt`.
2. In `packaging/aur/PKGBUILD`, set `pkgver=0.1.0` and replace `sha256sums=('SKIP')` with the real hash.
3. Regenerate `.SRCINFO`:
   ```sh
   makepkg --printsrcinfo > packaging/aur/.SRCINFO
   ```
4. Push both files to the `fnclaude-bin` AUR git repo.

## 3. winget

After the GitHub Release is up:

1. In `packaging/winget/`, bump `PackageVersion` in all three manifests.
2. In `installer.yaml`, update `InstallerUrl` and `InstallerSha256` (sha256 of the `.zip`, uppercase hex).
3. Fork `microsoft/winget-pkgs`, copy the three manifests into:
   ```
   manifests/f/fnrhombus/fnclaude/<version>/
   ```
   following the `fnrhombus.fnclaude` identifier path.
4. Open a PR against `microsoft/winget-pkgs`.
