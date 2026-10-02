package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
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
	modeFolders
)

// item adapts a Note to the bubbles list.
type item struct{ Note }

func (i item) Title() string       { return i.Note.Title }
func (i item) Description() string { return describe(i.Note) }
func (i item) FilterValue() string { return i.Note.Title + " " + i.Note.Body }

// folderItem adapts a Folder to the bubbles list.
type folderItem struct{ Folder }

func (f folderItem) Title() string       { return f.Folder.Name }
func (f folderItem) FilterValue() string { return f.Folder.Name }
func (f folderItem) Description() string {
	s := fmt.Sprintf("%d notes", f.Count)
	if f.Count == 1 {
		s = "1 note"
	}
	if !f.Time.IsZero() {
		s += " · " + humanTime(f.Time)
	}
	return s
}

type reloadMsg struct {
	dir        string
	stamp      string // dirStamp taken just before reading
	notes      []Note
	err        error
	selectPath string
	fromPoll   bool // reloaded because the folder changed on disk
}

// pollMsg asks Update to check the folder on disk for changes made elsewhere
// (voice notes, another editor). Polling keeps it dependency-free and cheap:
// one directory listing a second.
type pollMsg struct{}

const pollInterval = time.Second

func pollCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollMsg{} })
}

type foldersMsg struct {
	folders []Folder
	err     error
}

type editorDoneMsg struct{ err error }

type clearStatusMsg struct{ id int }

type model struct {
	root  string // the notes directory; its subdirectories are folders
	dir   string // folder currently browsed: root or one of its subdirectories
	popup bool   // started with `jotter new`: exit once the note is saved or cancelled
	mode  mode

	folders list.Model // folder picker, shown at launch when root has subfolders
	list    list.Model
	view    viewport.Model
	ta      textarea.Model

	width, height int
	focusPreview  bool
	previewPath   string
	previewBody   string

	stamp string // dirStamp/foldersStamp of what is shown, to spot changes on disk

	status    string
	statusErr bool
	statusID  int

	discardArmed  bool
	pendingDelete *Note

	openPath string // started with `jotter open <note>`: select it once the list loads

	editPath string // note being edited in the textarea; empty when composing a new one
	editOrig string // its text when editing started, to detect unsaved changes

	// selectAll is ctrl+a in the editor: the whole note is highlighted, and the
	// next key copies, cuts, deletes or replaces it. The textarea has no
	// selection of its own, so this is the only kind there is.
	selectAll bool
	taStyle   textarea.Style // the textarea's normal focused style, restored on deselect
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

	f := list.New(nil, d, 0, 0)
	f.Title = "Folders"
	f.Styles.Title = l.Styles.Title
	f.SetShowHelp(false)
	f.SetFilteringEnabled(false)
	f.SetStatusBarItemName("folder", "folders")
	f.DisableQuitKeybindings()

	m := model{root: dir, dir: dir, popup: popup, folders: f, list: l, view: viewport.New(0, 0), ta: ta, taStyle: ta.FocusedStyle}
	if popup {
		m.mode = modeCompose
		m.ta.Focus()
	}
	return m
}

func (m model) Init() tea.Cmd {
	switch {
	case m.popup:
		return textarea.Blink
	case m.openPath == "" && hasSubfolders(m.root):
		return tea.Batch(m.showFolders(), pollCmd())
	}
	return tea.Batch(loadCmd(m.dir, m.openPath), pollCmd())
}

// showFolders loads the folder list; the picker opens when it arrives.
func (m model) showFolders() tea.Cmd {
	root := m.root
	return func() tea.Msg {
		folders, err := loadFolders(root)
		return foldersMsg{folders, err}
	}
}

// openFolder leaves the picker and browses dir.
func (m model) openFolder(dir string) (tea.Model, tea.Cmd) {
	m.mode = modeList
	m.dir = dir
	m.stamp = dirStamp(dir)
	m.previewPath = ""
	m.list.ResetFilter()
	m.list.ResetSelected()
	m.list.SetItems(nil)
	m.list.Title = "Notes"
	if dir != m.root {
		m.list.Title = "Notes / " + filepath.Base(dir)
	}
	return m, loadCmd(dir, "")
}

func loadCmd(dir, selectPath string) tea.Cmd {
	return func() tea.Msg {
		stamp := dirStamp(dir)
		notes, err := loadNotes(dir)
		return reloadMsg{dir: dir, stamp: stamp, notes: notes, err: err, selectPath: selectPath}
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
		if msg.dir != m.dir {
			return m, nil // a load for a folder that has since been left
		}
		if msg.err != nil {
			return m, m.setStatus(msg.err.Error(), true)
		}
		m.stamp = msg.stamp
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = item{n}
		}
		var added *Note
		if msg.fromPoll {
			added = newestAdded(m.list.Items(), msg.notes)
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
		m.refreshPreview(false)
		if added != nil {
			cmd = tea.Batch(cmd, m.setStatus("New note: "+added.Title, false))
		}
		return m, cmd

	case pollMsg:
		if m.popup {
			return m, nil
		}
		var cmd tea.Cmd
		switch m.mode {
		case modeList:
			if s := dirStamp(m.dir); s != m.stamp {
				m.stamp = s
				load := loadCmd(m.dir, m.selectedPath())
				cmd = func() tea.Msg {
					msg := load().(reloadMsg)
					msg.fromPoll = true
					return msg
				}
			}
		case modeFolders:
			if s := foldersStamp(m.root); s != m.stamp {
				m.stamp = s
				cmd = m.showFolders()
			}
		}
		// Composing or confirming a delete: leave the list alone and look
		// again afterwards; the stamp is unchanged so the change still shows.
		return m, tea.Batch(cmd, pollCmd())

	case foldersMsg:
		if msg.err != nil {
			return m, m.setStatus(msg.err.Error(), true)
		}
		// Coming back from a folder, select it; refreshing the picker in
		// place, keep whatever is selected.
		want := m.dir
		if m.mode == modeFolders {
			if f, ok := m.folders.SelectedItem().(folderItem); ok {
				want = f.Path
			}
		} else {
			m.stamp = foldersStamp(m.root)
		}
		m.mode = modeFolders
		items := make([]list.Item, len(msg.folders))
		sel := 0
		for i, f := range msg.folders {
			items[i] = folderItem{f}
			if f.Path == want {
				sel = i
			}
		}
		cmd := m.folders.SetItems(items)
		m.folders.Select(sel)
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
		case modeFolders:
			return m.updateFolders(msg)
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
	case "f":
		if hasSubfolders(m.root) {
			return m, m.showFolders()
		}
		return m, nil
	case "esc":
		// esc clears an applied search first; with none, it goes back to folders.
		if m.list.FilterState() == list.Unfiltered && hasSubfolders(m.root) {
			return m, m.showFolders()
		}
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

func (m model) updateFolders(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "enter", "l", "right":
		if f, ok := m.folders.SelectedItem().(folderItem); ok {
			return m.openFolder(f.Path)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.folders, cmd = m.folders.Update(msg)
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
	m.setSelectAll(false)
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

// setSelectAll highlights the whole textarea, or puts its normal style back.
func (m *model) setSelectAll(on bool) {
	m.selectAll = on
	m.ta.FocusedStyle = m.taStyle
	if on {
		hl := lipgloss.NewStyle().Background(accent).Foreground(lipgloss.Color("0"))
		m.ta.FocusedStyle.Text = hl
		m.ta.FocusedStyle.CursorLine = hl
	}
	// The textarea renders through a pointer that Focus takes to FocusedStyle,
	// so the new style only shows after refocusing.
	if m.ta.Focused() {
		m.ta.Focus()
	}
}

// updateSelectAll handles the key after ctrl+a. handled is false when the key
// should still reach the textarea (typing or pasting over the selection).
func (m *model) updateSelectAll(msg tea.KeyMsg) (cmd tea.Cmd, handled bool) {
	switch msg.String() {
	case "ctrl+a":
		return nil, true
	case "ctrl+c", "ctrl+x":
		cmd = m.setStatus("Copied", false)
		if err := clipboard.WriteAll(m.ta.Value()); err != nil {
			cmd = m.setStatus("copy: "+err.Error(), true)
		} else if msg.String() == "ctrl+x" {
			m.ta.Reset()
			cmd = m.setStatus("Cut", false)
		}
		m.setSelectAll(false)
		return cmd, true
	case "backspace", "delete":
		m.ta.Reset()
		m.setSelectAll(false)
		return nil, true
	}
	m.setSelectAll(false)
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace || msg.Type == tea.KeyEnter || msg.String() == "ctrl+v" {
		m.ta.Reset() // replace the selection with what's typed or pasted
		return nil, false
	}
	// Anything else (arrows, esc, ctrl+s, ...) just drops the selection; esc
	// stops there, other keys carry on as usual.
	return nil, msg.Type == tea.KeyEsc
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectAll {
		if cmd, handled := m.updateSelectAll(msg); handled {
			return m, cmd
		}
	}

	switch msg.String() {
	case "ctrl+a":
		if m.ta.Value() != "" {
			m.setSelectAll(true)
		}
		return m, nil

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

// openEditor runs $VISUAL or $EDITOR on path, suspending the TUI meanwhile.
// With neither set it falls back to the first of nvim, vim, nano, vi found.
func openEditor(path string) tea.Cmd {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		for _, e := range []string{"nvim", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return func() tea.Msg { return editorDoneMsg{errors.New("no editor found; set $EDITOR")} }
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
	m.folders.SetSize(m.width, bodyH)
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
	if !force && n.Path == m.previewPath && n.Body == m.previewBody {
		return
	}
	if n.Path != m.previewPath {
		m.view.GotoTop()
	}
	m.previewPath, m.previewBody = n.Path, n.Body
	m.view.SetContent(renderMarkdown(n.Body, m.view.Width))
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
	if m.mode == modeFolders {
		body := lipgloss.NewStyle().Width(m.width).Height(bodyH).MaxHeight(bodyH).Render(m.folders.View())
		return lipgloss.JoinVertical(lipgloss.Left, body, m.footer())
	}
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
		lipgloss.NewStyle().Foreground(dim).Render("ctrl+s save · ctrl+a select all · esc cancel"),
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
	case m.mode == modeFolders:
		return style.Foreground(dim).Render("enter open · ↑ ↓ move · q quit")
	case m.focusPreview:
		return style.Foreground(dim).Render("↑ ↓ pgup pgdn scroll · tab back to list")
	default:
		return style.Foreground(dim).Render(m.listHelp())
	}
}

func (m model) listHelp() string {
	h := "n new · enter edit · o open in editor · d delete · / search · tab preview"
	if hasSubfolders(m.root) {
		h += " · f folders"
	}
	return h + " · q quit"
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

// newestAdded returns the newest note in notes that was not among old, or nil.
// notes is newest first, so the first unseen one is it.
func newestAdded(old []list.Item, notes []Note) *Note {
	seen := make(map[string]bool, len(old))
	for _, it := range old {
		if n, ok := it.(item); ok {
			seen[n.Path] = true
		}
	}
	for i := range notes {
		if !seen[notes[i].Path] {
			return &notes[i]
		}
	}
	return nil
}
