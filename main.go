// jotter is a small TUI for plain-markdown notes stored one-per-file in ~/Notes.
//
//	jotter              open the browser (list + preview)
//	jotter new          open straight into a new note; saving or cancelling exits
//	jotter open <note>  open the browser with that note selected
//
// Set JOTTER_DIR (or NOTES_DIR) to use a different directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `usage: jotter [new | open <note>]

  jotter              browse notes: list on the left, preview on the right
  jotter new          write a new note; saving or cancelling exits
  jotter open <note>  browse with <note> selected (a path or a file name)

Notes are markdown files in $JOTTER_DIR (default ~/Notes).`

func main() {
	popup := false
	open := ""
	switch {
	case len(os.Args) == 1:
	case len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help"):
		fmt.Println(usage)
		return
	case len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version"):
		fmt.Println("jotter", version)
		return
	case len(os.Args) == 2 && os.Args[1] == "new":
		popup = true
	case len(os.Args) == 3 && os.Args[1] == "open":
		open = os.Args[2]
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	dir, err := notesDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jotter:", err)
		os.Exit(1)
	}

	m := newModel(dir, popup)
	if open != "" {
		path := findNote(dir, open)
		// The browser starts in the note's folder (the top level or one below it).
		if folder := filepath.Dir(path); folder != dir {
			m.dir = folder
			m.list.Title = "Notes / " + filepath.Base(folder)
		}
		m.openPath = path
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "jotter:", err)
		os.Exit(1)
	}
}
