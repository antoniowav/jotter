// notes is a small TUI for plain-markdown notes stored one-per-file in ~/Notes.
//
//	notes              open the browser (list + preview)
//	notes new          open straight into a new note; saving or cancelling exits
//	notes open <note>  open the browser with that note selected
//
// Set NOTES_DIR to use a different directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	popup := false
	open := ""
	switch {
	case len(os.Args) == 1:
	case len(os.Args) == 2 && os.Args[1] == "new":
		popup = true
	case len(os.Args) == 3 && os.Args[1] == "open":
		open = os.Args[2]
	default:
		fmt.Fprintln(os.Stderr, "usage: notes [new | open <note>]")
		os.Exit(2)
	}

	dir, err := notesDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}

	m := newModel(dir, popup)
	if open != "" {
		// Notes are listed by their path inside the notes directory, so a bare
		// file name or a path from elsewhere is matched by its base name.
		m.openPath = filepath.Join(dir, filepath.Base(open))
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}
