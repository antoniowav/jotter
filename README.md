# notes

Tiny terminal notes app. One markdown file per note in `~/Notes`
(override with `NOTES_DIR`). Voice notes dictated with Super+H land in the
same folder with a `-voice` suffix (see `~/.local/bin/voice-note-stamp`).

## Usage

    notes        browse: list on the left, preview on the right
    notes new    jump straight into a new note; saving or cancelling exits
    notes open <note>   browse with that note selected (path or file name)

Keys in the browser: `n` new, `enter` edit in place, `o` open in `$EDITOR`,
`d` delete, `/` search, `tab` focus the preview to scroll it, `r` reload,
`q` quit. While writing: `ctrl+s` saves, `esc` cancels (twice if there are
unsaved changes).
The first line of a note is its title.

## Desktop integration (Omarchy / Hyprland)

- Super+N opens the browser in a terminal.
- Super+Alt+N opens a small floating terminal (class `notes-popup`) running `notes new`.

Bindings live in `~/.config/hypr/bindings.lua`, the float rule in
`~/.config/hypr/hyprland.lua`.

## Build

    go build -o ~/.local/bin/notes .
