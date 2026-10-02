package main

import (
	"errors"
	"fmt"
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

// findNote resolves the argument of `notes open` to a note path: an existing
// file in root or one of its folders as given, otherwise the first note with
// that file name, looking in root first and then in each folder. A name that
// matches nothing resolves to root, where the browser starts as usual.
func findNote(root, arg string) string {
	if abs, err := filepath.Abs(arg); err == nil {
		if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") &&
			strings.Count(rel, string(filepath.Separator)) <= 1 {
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				return abs
			}
		}
	}
	base := filepath.Base(arg)
	if !strings.HasSuffix(base, ".md") {
		base += ".md"
	}
	dirs := []string{root}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(d, base)); err == nil {
			return filepath.Join(d, base)
		}
	}
	return filepath.Join(root, base)
}

// Folder is a directory of notes: the notes directory itself or one of its
// direct subdirectories.
type Folder struct {
	Path  string
	Name  string
	Count int
	Time  time.Time // newest note in it, zero when empty
}

// loadFolders lists root followed by its subdirectories, alphabetically.
// Only one level is shown; deeper directories are left alone.
func loadFolders(root string) ([]Folder, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	folders := []Folder{folderOf(root, "Notes")}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			folders = append(folders, folderOf(filepath.Join(root, e.Name()), e.Name()))
		}
	}
	sort.SliceStable(folders[1:], func(i, j int) bool {
		return strings.ToLower(folders[1+i].Name) < strings.ToLower(folders[1+j].Name)
	})
	return folders, nil
}

func folderOf(path, name string) Folder {
	f := Folder{Path: path, Name: name}
	if notes, err := loadNotes(path); err == nil {
		f.Count = len(notes)
		if len(notes) > 0 {
			f.Time = notes[0].Time
		}
	}
	return f
}

// hasSubfolders reports whether root contains any non-hidden directory.
func hasSubfolders(root string) bool {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			return true
		}
	}
	return false
}

// dirStamp summarises the notes in dir (names, sizes, mtimes) so a change made
// by another program can be spotted without reloading every file.
func dirStamp(dir string) string {
	entries, _ := os.ReadDir(dir)
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if info, err := e.Info(); err == nil {
			fmt.Fprintf(&b, "%s %d %d\n", name, info.Size(), info.ModTime().UnixNano())
		}
	}
	return b.String()
}

// foldersStamp is dirStamp over root and every folder in it, plus the folder
// names, so the picker notices new folders as well as new notes.
func foldersStamp(root string) string {
	var b strings.Builder
	b.WriteString(dirStamp(root))
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			b.WriteString("/" + e.Name() + "\n" + dirStamp(filepath.Join(root, e.Name())))
		}
	}
	return b.String()
}
