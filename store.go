package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// tsLayout is the timestamp prefix used for note file names.
const tsLayout = "2006-01-02-150405"

// Note is one markdown file in the notes directory.
type Note struct {
	Path  string
	Title string
	Body  string
	Time  time.Time
	Voice bool // dictated via voice-note (file name ends in -voice)
}

// notesDir returns the notes directory, creating it if needed.
func notesDir() (string, error) {
	dir := os.Getenv("NOTES_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "Notes")
	}
	return dir, os.MkdirAll(dir, 0o755)
}

// loadNotes reads every *.md file in dir, newest first.
func loadNotes(dir string) ([]Note, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		body := string(data)
		notes = append(notes, Note{
			Path:  path,
			Title: titleOf(body, base),
			Body:  body,
			Time:  noteTime(base, info.ModTime()),
			Voice: strings.HasSuffix(base, "-voice"),
		})
	}
	sort.SliceStable(notes, func(i, j int) bool { return notes[i].Time.After(notes[j].Time) })
	return notes, nil
}

// noteTime parses the timestamp prefix of a file name, falling back to mtime.
func noteTime(base string, fallback time.Time) time.Time {
	if len(base) >= len(tsLayout) {
		if t, err := time.ParseInLocation(tsLayout, base[:len(tsLayout)], time.Local); err == nil {
			return t
		}
	}
	return fallback
}

// titleOf is the first non-empty line of body without markdown heading marks.
func titleOf(body, fallback string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > 80 {
			line = string(r[:79]) + "…"
		}
		return line
	}
	return fallback
}

// saveNote writes text as a new timestamped note and returns its path.
func saveNote(dir, text string) (string, error) {
	text = strings.TrimRight(text, " \t\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("empty note")
	}
	path := newNotePath(dir)
	return path, os.WriteFile(path, []byte(text+"\n"), 0o644)
}

// newNotePath returns an unused timestamped path in dir.
func newNotePath(dir string) string {
	now := time.Now()
	for i := 0; ; i++ {
		path := filepath.Join(dir, now.Add(time.Duration(i)*time.Second).Format(tsLayout)+".md")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return path
		}
	}
}
