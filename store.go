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

// notesDir returns the notes directory, creating it if needed: $JOTTER_DIR,
// else $NOTES_DIR, else ~/Notes.
func notesDir() (string, error) {
	dir := os.Getenv("JOTTER_DIR")
	if dir == "" {
		dir = os.Getenv("NOTES_DIR")
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "Notes")
	}
	// Notes are private: a new notes directory is only readable by you.
	return dir, os.MkdirAll(dir, 0o700) // #nosec G703 -- the directory the user chose
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
		data, err := os.ReadFile(path) // #nosec G304 -- notes are whatever *.md files the user keeps here
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
	for _, line := range strings.Split(sanitize(body), "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > 80 {
			line = string(r[:79]) + "…"
		}
		return line
	}
	return sanitize(fallback)
}

// sanitize drops control characters other than newline and tab. Notes can come
// from anywhere (other programs, synced folders), and escape sequences printed
// raw would reach the terminal: retitling it, writing to the clipboard, or
// redrawing the screen.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || (r >= 0x7f && r < 0xa0) {
			return -1
		}
		return r
	}, s)
}

// saveNote writes text as a new timestamped note and returns its path. The file
// is created exclusively (never overwriting a note saved in the same second by
// another program) and readable only by you.
func saveNote(dir, text string) (string, error) {
	text = strings.TrimRight(text, " \t\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("empty note")
	}
	now := time.Now()
	for i := 0; i < 1000; i++ {
		path := filepath.Join(dir, now.Add(time.Duration(i)*time.Second).Format(tsLayout)+".md")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- a new file in the notes directory
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.WriteString(text + "\n"); err != nil {
			_ = f.Close()
			return "", err
		}
		return path, f.Close()
	}
	return "", errors.New("no free file name for the note")
}

// findNote resolves the argument of `jotter open` to a note path: an existing
// file in root or one of its folders as given, otherwise the first note with
// that file name, looking in root first and then in each folder. A name that
// matches nothing resolves to root, where the browser starts as usual.
func findNote(root, arg string) string {
	if abs, err := filepath.Abs(arg); err == nil {
		if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") &&
			strings.Count(rel, string(filepath.Separator)) <= 1 {
			if info, err := os.Stat(abs); err == nil && !info.IsDir() { // #nosec G703 -- a path the user typed
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
	f := Folder{Path: path, Name: sanitize(name)}
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
