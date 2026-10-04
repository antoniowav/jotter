# Changelog

All notable changes to jotter are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- The window title is "Jotter", so bars and window switchers show it instead of the terminal's name.

### Fixed
- `jotter --version` printed `dev` when installed with `go install`.

## [0.1.0] - 2026-10-04

First public release.

### Added
- Browser with a note list, search and a rendered markdown preview.
- Write and edit notes in place, or open them in `$VISUAL` / `$EDITOR`.
- `jotter new` for quick capture, and `jotter open <note>` to start on a note.
- Folders (one level of subfolders), with a folder picker at launch.
- Notes changed by other programs show up live.
- Select all in the editor (`ctrl+a`) to copy, cut or replace the whole note.
- `--version` and `--help`.
- `JOTTER_DIR` to choose the notes folder (`NOTES_DIR` still works).
- `install` / `uninstall` scripts for `~/.local`, a desktop entry, an icon and an Arch PKGBUILD.

### Changed (since the private builds)
- Renamed from `notes` to `jotter`: `notes` is already taken on the AUR.

### Fixed (since the private builds)
- `jotter open` now finds notes inside folders, not only at the top level.
- `o` no longer assumes nvim: it uses `$VISUAL`, `$EDITOR`, or the first of nvim, vim, nano or vi.

### Security
- Control characters in notes are no longer sent to the terminal. A crafted note could retitle the terminal
  or silently replace the clipboard (OSC 52) just by being shown.
- New notes are created `600` and a new notes folder `700`, instead of world-readable.
- New notes are created exclusively, so a note saved in the same second by another program is never overwritten.
- `golang.org/x/sys` updated to v0.44.0 (GO-2026-5024; not reachable from jotter's code). Go ≥ 1.25 is now needed to build.
- CI: tests, govulncheck, staticcheck, gosec, shellcheck, gitleaks, namcap, and CodeQL once public.

[Unreleased]: https://github.com/antoniowav/jotter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/antoniowav/jotter/releases/tag/v0.1.0
