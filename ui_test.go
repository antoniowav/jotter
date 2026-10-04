package main

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWindowTitle(t *testing.T) {
	long := strings.Repeat("x", 60)
	compose := func(n *Note) model {
		m, _ := newModel(t.TempDir(), false).startCompose(n)
		return m.(model)
	}
	for _, c := range []struct {
		name string
		m    model
		want string
	}{
		{"browsing", newModel(t.TempDir(), false), "Jotter"},
		{"jotter new", newModel(t.TempDir(), true), "Jotter — New note"},
		{"new note", compose(nil), "Jotter — New note"},
		{"editing", compose(&Note{Path: "a.md", Title: "Shopping list"}), "Jotter — Editing: Shopping list"},
		{"long title", compose(&Note{Path: "a.md", Title: long}), "Jotter — Editing: " + long[:39] + "…"},
		{"escapes", compose(&Note{Path: "a.md", Title: "a\x1b]0;x\x07b"}), "Jotter — Editing: a]0;xb"},
	} {
		if got, want := c.m.windowTitle()(), tea.SetWindowTitle(c.want)(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: title %v, want %q", c.name, got, c.want)
		}
	}
}
