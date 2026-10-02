# Changelog

All notable changes to notes are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-02

First public release.

### Added
- Browser with a note list, search and a rendered markdown preview.
- Write and edit notes in place, or open them in `$VISUAL` / `$EDITOR`.
- `notes new` for quick capture, and `notes open <note>` to start on a note.
- Folders (one level of subfolders), with a folder picker at launch.
- Notes changed by other programs show up live.
- Select all in the editor (`ctrl+a`) to copy, cut or replace the whole note.
- `--version` and `--help`.
- `install` / `uninstall` scripts for `~/.local`, a desktop entry, an icon and an Arch PKGBUILD.

### Fixed (since the private builds)
- `notes open` now finds notes inside folders, not only at the top level.
- `o` no longer assumes nvim: it uses `$VISUAL`, `$EDITOR`, or the first of nvim, vim, nano or vi.

[Unreleased]: https://github.com/antoniowav/notes/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/antoniowav/notes/releases/tag/v0.1.0
