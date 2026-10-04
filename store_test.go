package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindNote(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "Work"), 0o755)
	top := filepath.Join(root, "2026-01-01-100000.md")
	sub := filepath.Join(root, "Work", "2026-01-02-100000.md")
	os.WriteFile(top, []byte("top\n"), 0o644)
	os.WriteFile(sub, []byte("sub\n"), 0o644)

	for arg, want := range map[string]string{
		top:                               top,
		sub:                               sub,
		"2026-01-01-100000.md":            top,
		"2026-01-02-100000.md":            sub,
		"2026-01-02-100000":               sub,
		"/elsewhere/2026-01-02-100000.md": sub,
		"missing.md":                      filepath.Join(root, "missing.md"),
	} {
		if got := findNote(root, arg); got != want {
			t.Errorf("findNote(%q) = %q, want %q", arg, got, want)
		}
	}
}

func TestTitleOf(t *testing.T) {
	for body, want := range map[string]string{
		"# Heading\nbody":  "Heading",
		"\n\n  plain line": "plain line",
		"":                 "fallback",
	} {
		if got := titleOf(body, "fallback"); got != want {
			t.Errorf("titleOf(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestSanitize(t *testing.T) {
	in := "milk \x1b]0;PWNED\x07 eggs\r\n\x1b]52;c;ZWNobyBoaQ==\x07\tbread\u009b"
	want := "milk ]0;PWNED eggs\n]52;c;ZWNobyBoaQ==\tbread"
	if got := sanitize(in); got != want {
		t.Errorf("sanitize = %q, want %q", got, want)
	}
}

func TestSaveNotePrivateAndExclusive(t *testing.T) {
	dir := t.TempDir()
	a, err := saveNote(dir, "first")
	if err != nil {
		t.Fatal(err)
	}
	b, err := saveNote(dir, "second") // same second: must not overwrite the first
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("both notes saved to %s", a)
	}
	info, _ := os.Stat(a)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("note mode = %o, want 600", perm)
	}
	if body, _ := os.ReadFile(a); string(body) != "first\n" {
		t.Errorf("first note = %q", body)
	}
}
