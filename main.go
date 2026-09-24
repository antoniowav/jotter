// notes is a small TUI for plain-markdown notes stored one-per-file in ~/Notes.
//
//	notes       open the browser (list + preview)
//	notes new   open straight into a new note; saving or cancelling exits
//
// Set NOTES_DIR to use a different directory.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	popup := false
	switch {
	case len(os.Args) == 1:
	case len(os.Args) == 2 && os.Args[1] == "new":
		popup = true
	default:
		fmt.Fprintln(os.Stderr, "usage: notes [new]")
		os.Exit(2)
	}

	dir, err := notesDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(dir, popup), tea.WithAltScreen(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}
