# notes

**A tiny terminal notes app: plain markdown files, folders, a live preview and nothing to lock you in.**

<!-- Screenshots: add the images to docs/screenshots/ and uncomment.
![The browser: list on the left, rendered preview on the right](docs/screenshots/browser.png)
![Writing a note](docs/screenshots/compose.png)
![The folder picker](docs/screenshots/folders.png)
-->
> 📸 _Screenshots coming soon: browser · writing · folders._

Every note is a `.md` file in one folder, so they work with any editor, sync tool,
`grep` or git. `notes` gives you a fast way to browse, search, read and write them.

## Features

- **Browse** with a list on the left and a rendered markdown preview on the right (headings, lists, quotes, code, bold, rules).
- **Write in place**, or open the note in your own `$EDITOR`.
- **Search** across titles and full text.
- **Folders**: subfolders of your notes directory show up in a folder picker.
- **Live**: notes added or changed by other programs appear within a second.
- **Quick capture**: `notes new` opens straight into a blank note and exits on save. Bind it to a hotkey.
- **Native colours**: it uses your terminal's ANSI palette, so it matches any theme (including Omarchy's).

## Install

### Arch Linux (AUR)

```sh
yay -S notes          # or: paru -S notes
```

### From a git clone (any Linux)

Needs Go ≥ 1.24.

```sh
git clone https://github.com/antoniowav/notes.git
cd notes
./install             # builds and installs into ~/.local
```

`./install` puts the binary in `~/.local/bin` and adds a desktop entry and icon.
Use `PREFIX=/some/where ./install` to install elsewhere. Update with `git pull && ./install`.

## Usage

```sh
notes                   # browse
notes new               # write a new note; saving or cancelling exits
notes open <note>       # browse with that note selected (a path, or just its file name)
notes --version
```

The first line of a note is its title (leading `#` marks are dropped).

### Keys: browsing

| Key | Action |
| --- | --- |
| <kbd>↑</kbd> <kbd>↓</kbd> / <kbd>k</kbd> <kbd>j</kbd> | Move |
| <kbd>←</kbd> <kbd>→</kbd> / <kbd>h</kbd> <kbd>l</kbd> / <kbd>PgUp</kbd> <kbd>PgDn</kbd> / <kbd>b</kbd> <kbd>u</kbd> | Previous / next page |
| <kbd>g</kbd> <kbd>G</kbd> / <kbd>Home</kbd> <kbd>End</kbd> | First / last note |
| <kbd>n</kbd> | New note |
| <kbd>Enter</kbd> / <kbd>e</kbd> | Edit the note in place |
| <kbd>o</kbd> | Open the note in `$VISUAL` / `$EDITOR` (else nvim, vim, nano or vi) |
| <kbd>d</kbd> | Delete the note (asks first: <kbd>y</kbd> confirms, any other key cancels) |
| <kbd>/</kbd> | Search (type, then <kbd>Enter</kbd> to keep the results or <kbd>Esc</kbd> to cancel) |
| <kbd>Esc</kbd> | Clear the search; with no search, back to the folders |
| <kbd>Tab</kbd> | Focus the preview to scroll it |
| <kbd>r</kbd> | Reload |
| <kbd>f</kbd> | Folders (when there are any) |
| <kbd>q</kbd> / <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit |

### Keys: preview focused

| Key | Action |
| --- | --- |
| <kbd>↑</kbd> <kbd>↓</kbd> / <kbd>k</kbd> <kbd>j</kbd> | Scroll a line |
| <kbd>PgDn</kbd> / <kbd>Space</kbd> / <kbd>f</kbd>, <kbd>PgUp</kbd> / <kbd>b</kbd> | Scroll a page |
| <kbd>d</kbd> / <kbd>Ctrl</kbd>+<kbd>D</kbd>, <kbd>u</kbd> / <kbd>Ctrl</kbd>+<kbd>U</kbd> | Scroll half a page |
| <kbd>Tab</kbd> / <kbd>Esc</kbd> / <kbd>q</kbd> | Back to the list |

### Keys: folder picker

| Key | Action |
| --- | --- |
| <kbd>↑</kbd> <kbd>↓</kbd> / <kbd>k</kbd> <kbd>j</kbd>, <kbd>g</kbd> <kbd>G</kbd> | Move |
| <kbd>Enter</kbd> / <kbd>l</kbd> / <kbd>→</kbd> | Open the folder |
| <kbd>q</kbd> / <kbd>Esc</kbd> / <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit |

### Keys: writing

| Key | Action |
| --- | --- |
| <kbd>Ctrl</kbd>+<kbd>S</kbd> | Save |
| <kbd>Esc</kbd> | Cancel (press twice if there are unsaved changes) |
| <kbd>Ctrl</kbd>+<kbd>A</kbd> | Select the whole note; then <kbd>Ctrl</kbd>+<kbd>C</kbd> copies, <kbd>Ctrl</kbd>+<kbd>X</kbd> cuts, <kbd>Backspace</kbd>/<kbd>Delete</kbd> clears, typing or <kbd>Ctrl</kbd>+<kbd>V</kbd> replaces it |
| <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit **without saving** (when nothing is selected) |
| <kbd>Ctrl</kbd>+<kbd>V</kbd> | Paste |
| <kbd>Home</kbd> <kbd>End</kbd> / <kbd>Ctrl</kbd>+<kbd>E</kbd> | Start / end of the line |
| <kbd>Alt</kbd>+<kbd>←</kbd> <kbd>→</kbd> | Word left / right |
| <kbd>Ctrl</kbd>+<kbd>Home</kbd> <kbd>End</kbd> | Start / end of the note |
| <kbd>Ctrl</kbd>+<kbd>W</kbd> / <kbd>Alt</kbd>+<kbd>Backspace</kbd> | Delete the word before the cursor |
| <kbd>Ctrl</kbd>+<kbd>K</kbd> / <kbd>Ctrl</kbd>+<kbd>U</kbd> | Delete to the end / start of the line |

## Requirements

- Linux (or any Unix) and a terminal.
- Go ≥ 1.24 to build it (not needed with the AUR package at runtime).
- _Optional_: `wl-clipboard` (Wayland) or `xclip`/`xsel` (X11) for copy and cut.
- _Optional_: an editor in `$VISUAL` / `$EDITOR` for <kbd>o</kbd>.

## Where your notes are stored

- In **`$NOTES_DIR`**, or **`~/Notes`** when it isn't set. The folder is created on first run.
- One file per note, named by when it was written: `YYYY-MM-DD-HHMMSS.md` (e.g. `2026-10-02-194421.md`).
  Notes are sorted by that time; files with other names use their modification time.
- Any `*.md` file you put there yourself shows up too. Hidden files and non-`.md` files are ignored.
- **Folders** are the direct subfolders of the notes directory (one level; deeper folders aren't shown).
  `notes new` always writes to the top level.
- A file name ending in `-voice.md` is labelled as a voice note. Handy if a dictation script drops notes in the same folder.
- **Deleting a note removes the file permanently.** There is no trash.

`notes` keeps no other state, config or cache.

## Desktop integration

Installing adds a *Notes* entry to your app launcher (it opens in your terminal).

On **Omarchy** you can bind it to keys, with a small floating window for quick capture:

```lua
-- ~/.config/hypr/bindings.lua
o.bind("SUPER + N", "Notes", "omarchy-launch-tui --app-id=org.omarchy.notes notes")
o.bind("SUPER + ALT + N", "New note", "omarchy-launch-tui --app-id=notes-popup notes new")

-- ~/.config/hypr/hyprland.lua
o.window("notes-popup", { float = true, center = true, size = { 800, 500 } })
```

Elsewhere, start `notes new` in a terminal with a class / app-id your window manager can float,
e.g. `foot --app-id=notes-popup notes new` or `alacritty --class notes-popup -e notes new`.

## Uninstall

- AUR / pacman: `sudo pacman -R notes`
- Git-clone install: `./uninstall` from the checkout

Your notes are never touched.

## License

[MIT](LICENSE) © Antonio Piattelli
