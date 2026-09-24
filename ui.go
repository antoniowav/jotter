package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeList mode = iota
	modeCompose
	modeConfirmDelete
)

// item adapts a Note to the bubbles list.
type item struct{ Note }

func (i item) Title() string       { return i.Note.Title }
func (i item) Description() string { return describe(i.Note) }
func (i item) FilterValue() string { return i.Note.Title + " " + i.Note.Body }

type reloadMsg struct {
	notes      []Note
	err        error
	selectPath string
}

type editorDoneMsg struct{ err error }

type clearStatusMsg struct{ id int }

type model struct {
	dir   string
	popup bool // started with `notes new`: exit once the note is saved or cancelled
	mode  mode

	list list.Model
	view viewport.Model
	ta   textarea.Model

	width, height int
	focusPreview  bool
	previewPath   string

	status    string
	statusErr bool
	statusID  int

	discardArmed  bool
	pendingDelete *Note

	editPath string // note being edited in the textarea; empty when composing a new one
	editOrig string // its text when editing started, to detect unsaved changes
}

func newModel(dir string, popup bool) model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Bold(true).Foreground(accent).BorderLeftForeground(accent)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(accent).BorderLeftForeground(accent)

	l := list.New(nil, d, 0, 0)
	l.Title = "Notes"
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(accent)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(accent)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(accent)
	l.FilterInput.PromptStyle = l.Styles.FilterPrompt
	l.FilterInput.Cursor.Style = l.Styles.FilterCursor
	l.SetShowHelp(false)
	l.SetStatusBarItemName("note", "notes")
	l.DisableQuitKeybindings()

	ta := textarea.New()
	ta.Placeholder = "Write your note…"
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	m := model{dir: dir, popup: popup, list: l, view: viewport.New(0, 0), ta: ta}
	if popup {
		m.mode = modeCompose
		m.ta.Focus()
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.popup {
		return textarea.Blink
	}
	return loadCmd(m.dir, "")
}

func loadCmd(dir, selectPath string) tea.Cmd {
	return func() tea.Msg {
		notes, err := loadNotes(dir)
		return reloadMsg{notes: notes, err: err, selectPath: selectPath}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.refreshPreview(true)
		return m, nil

	case reloadMsg:
		if msg.err != nil {
			return m, m.setStatus(msg.err.Error(), true)
		}
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = item{n}
		}
		cmd := m.list.SetItems(items)
		if msg.selectPath != "" {
			for i, it := range m.list.VisibleItems() {
				if n, ok := it.(item); ok && n.Path == msg.selectPath {
					m.list.Select(i)
					break
				}
			}
		}
		m.refreshPreview(true)
		return m, cmd

	case tea.FocusMsg:
		// Terminal regained focus: pick up notes edited elsewhere (GUI editor, voice-note).
		if m.mode == modeList {
			return m, loadCmd(m.dir, m.selectedPath())
		}
		return m, nil

	case editorDoneMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			cmd = m.setStatus("editor: "+msg.err.Error(), true)
		}
		return m, tea.Batch(cmd, loadCmd(m.dir, m.selectedPath()))

	case clearStatusMsg:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeCompose:
			return m.updateCompose(msg)
		case modeConfirmDelete:
			return m.updateConfirm(msg)
		default:
			return m.updateList(msg)
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	cmds = append(cmds, cmd)
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	m.refreshPreview(false)
	return m, tea.Batch(cmds...)
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While typing a filter, every key belongs to the list.
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.refreshPreview(false)
		return m, cmd
	}

	if m.focusPreview {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab", "esc", "q":
			m.focusPreview = false
			return m, nil
		}
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "n":
		return m.startCompose(nil)
	case "enter", "e":
		if n := m.selected(); n != nil {
			return m.startCompose(n)
		}
		return m, nil
	case "o":
		if n := m.selected(); n != nil {
			return m, openEditor(n.Path)
		}
		return m, nil
	case "d":
		if n := m.selected(); n != nil {
			m.mode = modeConfirmDelete
			m.pendingDelete = n
		}
		return m, nil
	case "r":
		return m, loadCmd(m.dir, m.selectedPath())
	case "tab":
		if m.showPreview() && m.selected() != nil {
			m.focusPreview = true
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshPreview(false)
	return m, cmd
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	n := m.pendingDelete
	m.pendingDelete = nil
	if n == nil || (msg.String() != "y" && msg.String() != "Y") {
		return m, nil
	}
	if err := os.Remove(n.Path); err != nil {
		return m, m.setStatus(err.Error(), true)
	}
	m.previewPath = ""
	return m, tea.Batch(m.setStatus("Deleted: "+n.Title, false), loadCmd(m.dir, ""))
}

// startCompose opens the textarea, empty for a new note or filled with n's text.
func (m model) startCompose(n *Note) (tea.Model, tea.Cmd) {
	m.mode = modeCompose
	m.discardArmed = false
	m.status = ""
	m.editPath, m.editOrig = "", ""
	m.ta.Reset()
	if n != nil {
		m.editPath = n.Path
		m.editOrig = strings.TrimRight(n.Body, "\n")
		m.ta.SetValue(m.editOrig)
	}
	return m, m.ta.Focus()
}

// saveCompose writes the textarea to the edited note, or to a new file.
func (m model) saveCompose() (string, error) {
	if m.editPath == "" {
		return saveNote(m.dir, m.ta.Value())
	}
	text := strings.TrimRight(m.ta.Value(), " \t\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("empty note (use d to delete it instead)")
	}
	return m.editPath, os.WriteFile(m.editPath, []byte(text+"\n"), 0o644)
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "ctrl+s":
		path, err := m.saveCompose()
		if err != nil {
			return m, m.setStatus(err.Error(), true)
		}
		if m.popup {
			return m, tea.Quit
		}
		m.mode = modeList
		m.ta.Blur()
		m.previewPath = ""
		return m, tea.Batch(m.setStatus("Saved", false), loadCmd(m.dir, path))

	case "esc":
		text := strings.TrimRight(m.ta.Value(), " \t\n")
		dirty := strings.TrimSpace(text) != "" && text != m.editOrig
		if dirty && !m.discardArmed {
			m.discardArmed = true
			m.statusID++
			m.status, m.statusErr = "Discard changes? Press esc again", true
			return m, nil
		}
		if m.popup {
			return m, tea.Quit
		}
		m.mode = modeList
		m.ta.Blur()
		m.status = ""
		m.discardArmed = false
		return m, nil
	}

	if m.discardArmed {
		m.discardArmed = false
		m.status = ""
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

// openEditor runs $EDITOR (default nvim) on path, suspending the TUI meanwhile.
func openEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "nvim"
	}
	c := exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDoneMsg{err} })
}

func (m *model) setStatus(s string, isErr bool) tea.Cmd {
	m.status, m.statusErr = s, isErr
	m.statusID++
	id := m.statusID
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{id} })
}

func (m model) selected() *Note {
	if it, ok := m.list.SelectedItem().(item); ok {
		n := it.Note
		return &n
	}
	return nil
}

func (m model) selectedPath() string {
	if n := m.selected(); n != nil {
		return n.Path
	}
	return ""
}

// Layout ---------------------------------------------------------------------

func (m model) showPreview() bool { return m.width >= 70 }

func (m model) bodyHeight() int { return max(m.height-1, 3) }

func (m model) listWidth() int {
	if !m.showPreview() {
		return m.width
	}
	return min(max(m.width*2/5, 28), 48)
}

func (m *model) layout() {
	if m.width == 0 {
		return
	}
	bodyH := m.bodyHeight()
	m.list.SetSize(m.listWidth(), bodyH)
	if m.showPreview() {
		m.view.Width = m.width - m.listWidth() - 3 // gap + border
		m.view.Height = max(bodyH-5, 1)            // border (2) + title, meta, blank (3)
	}
	m.ta.SetWidth(m.width)
	m.ta.SetHeight(max(m.height-3, 1))
}

func (m *model) refreshPreview(force bool) {
	n := m.selected()
	if n == nil {
		m.previewPath = ""
		m.view.SetContent("")
		return
	}
	if !force && n.Path == m.previewPath {
		return
	}
	m.previewPath = n.Path
	m.view.SetContent(renderMarkdown(n.Body, m.view.Width))
	m.view.GotoTop()
}

// View -----------------------------------------------------------------------

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.mode == modeCompose {
		return m.viewCompose()
	}
	bodyH := m.bodyHeight()
	body := lipgloss.NewStyle().Width(m.listWidth()).Height(bodyH).MaxHeight(bodyH).Render(m.list.View())
	if m.showPreview() {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", m.viewPreview())
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, m.footer())
}

func (m model) viewPreview() string {
	inner := m.view.Width
	innerH := m.view.Height + 3
	border := dim
	if m.focusPreview {
		border = accent
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Width(inner).Height(innerH).MaxHeight(innerH + 2)

	n := m.selected()
	if n == nil {
		return box.Render(lipgloss.NewStyle().Foreground(dim).Render("No note selected. Press n to write one."))
	}
	head := lipgloss.NewStyle().Bold(true).MaxWidth(inner).Render(n.Title) + "\n" +
		lipgloss.NewStyle().Foreground(dim).Render(describe(*n))
	return box.Render(head + "\n\n" + m.view.View())
}

func (m model) viewCompose() string {
	title := "New note"
	if m.editPath != "" {
		title = "Edit note"
	}
	head := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render(title),
		"  ",
		lipgloss.NewStyle().Foreground(dim).Render("ctrl+s save · esc cancel"),
	)
	return lipgloss.JoinVertical(lipgloss.Left, head, "", m.ta.View(), m.footer())
}

func (m model) footer() string {
	style := lipgloss.NewStyle().MaxWidth(m.width)
	switch {
	case m.mode == modeConfirmDelete && m.pendingDelete != nil:
		return style.Foreground(danger).Render(fmt.Sprintf("Delete %q? y / n", m.pendingDelete.Title))
	case m.status != "" && m.statusErr:
		return style.Foreground(danger).Render(m.status)
	case m.status != "":
		return style.Foreground(good).Render(m.status)
	case m.mode == modeCompose:
		return style.Foreground(dim).Render("First line becomes the title")
	case m.focusPreview:
		return style.Foreground(dim).Render("↑ ↓ pgup pgdn scroll · tab back to list")
	default:
		return style.Foreground(dim).Render("n new · enter edit · o open in $EDITOR · d delete · / search · tab preview · q quit")
	}
}

func describe(n Note) string {
	s := humanTime(n.Time)
	if n.Voice {
		s += " · voice"
	}
	return s
}

func humanTime(t time.Time) string {
	now := time.Now()
	day := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}
	switch days := int(day(now).Sub(day(t)).Hours() / 24); {
	case days == 0:
		return "Today " + t.Format("15:04")
	case days == 1:
		return "Yesterday " + t.Format("15:04")
	case t.Year() == now.Year():
		return t.Format("Jan 02 15:04")
	default:
		return t.Format("2006-01-02 15:04")
	}
}
