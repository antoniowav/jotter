# Release checklist

Replace `X.Y.Z` with the new version.

## 1. Prepare
- [ ] `VERSION` and `pkgver` in `packaging/arch/PKGBUILD` are `X.Y.Z` (and `pkgrel=1`).
- [ ] `CHANGELOG.md`: move *Unreleased* into `## [X.Y.Z] - YYYY-MM-DD` and update the compare links.
- [ ] `go mod tidy` leaves `go.mod`/`go.sum` unchanged; `go vet ./... && go test ./...` pass.

## 2. Test
- [ ] `./install`, then `jotter --version` prints `jotter X.Y.Z`.
- [ ] Browse, search, write, edit, delete a note; open one in `$EDITOR` with `o`.
- [ ] Folders: picker at launch, `f`/`esc` back to it, `jotter open <file in a folder>`.
- [ ] `jotter new` from the hotkey: saves and exits.
- [ ] Packaging: `cd packaging/arch && makepkg -f && namcap PKGBUILD *.pkg.tar.zst` (after step 3, with the tag pushed).

## 3. Tag and publish (only when the repo is meant to be public)
```sh
git commit -am "Release vX.Y.Z"
git tag -a vX.Y.Z -m "jotter X.Y.Z"
git push origin main --follow-tags
gh release create vX.Y.Z --title "jotter X.Y.Z" --notes "See CHANGELOG.md"
```

## 4. AUR
```sh
cd packaging/arch
updpkgsums                              # fills sha256sums from the GitHub tarball
makepkg -f && namcap PKGBUILD jotter-*.pkg.tar.zst
makepkg --printsrcinfo > .SRCINFO
git -C ../.. commit -am "PKGBUILD: checksums for vX.Y.Z" && git -C ../.. push

# the AUR package lives in its own git repo:
git clone ssh://aur@aur.archlinux.org/jotter.git ~/aur/jotter   # first time only
cp PKGBUILD .SRCINFO ~/aur/jotter/
cd ~/aur/jotter && git add PKGBUILD .SRCINFO && git commit -m "Update to X.Y.Z" && git push
```
