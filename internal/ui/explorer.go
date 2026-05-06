// internal/ui/explorer.go
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/Radashi/notas-tui/internal/notes"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Item wrapper ─────────────────────────────────────────────────────────────

type noteItem struct {
	note notes.Note
}

func (i noteItem) FilterValue() string { return i.note.Title }
func (i noteItem) Title() string       { return i.note.Title }
func (i noteItem) Description() string {
	size := fmt.Sprintf("%d bytes", i.note.Size)
	if i.note.Size > 1024 {
		size = fmt.Sprintf("%.1f kb", float64(i.note.Size)/1024)
	}
	return fmt.Sprintf("%s · %s", i.note.Modified.Format("02 Jan 15:04"), size)
}

// ── Delegate ──────────────────────────────────────────────────────────────────

type noteDelegate struct{}

func (d noteDelegate) Height() int                             { return 2 }
func (d noteDelegate) Spacing() int                            { return 1 }
func (d noteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d noteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(noteItem)
	if !ok {
		return
	}

	if index == m.Index() {
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).PaddingLeft(2)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(2)
		fmt.Fprintf(w, "%s\n%s", titleStyle.Render("▶ "+i.Title()), descStyle.Render(i.Description()))
	} else {
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).PaddingLeft(4)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(4)
		fmt.Fprintf(w, "%s\n%s", titleStyle.Render(i.Title()), descStyle.Render(i.Description()))
	}
}

// ── Estado ────────────────────────────────────────────────────────────────────

type explorerState int

const (
	stateList explorerState = iota
	stateCreating
	stateConfirmDelete
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Explorer struct {
	list    list.Model
	manager *notes.Manager
	input   textinput.Model
	state   explorerState
	err     error
	width   int
	height  int
}

func NewExplorer(manager *notes.Manager) Explorer {
	l := list.New([]list.Item{}, noteDelegate{}, 0, 0)
	l.Title = "Notas"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Padding(0, 0)
	// Deshabilitar el help automático de bubbles/list
	l.SetShowHelp(false)
	// Quitar el padding del área de items
	l.Styles.PaginationStyle = lipgloss.NewStyle().PaddingLeft(1)
	l.Styles.HelpStyle = lipgloss.NewStyle() // vacío, ya deshabilitamos el help

	ti := textinput.New()
	ti.Placeholder = "nombre-de-la-nota"
	ti.CharLimit = 60

	return Explorer{
		list:    l,
		manager: manager,
		input:   ti,
		state:   stateList,
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (e Explorer) Init() tea.Cmd {
	return loadNotesCmd(e.manager)
}

func loadNotesCmd(m *notes.Manager) tea.Cmd {
	return func() tea.Msg {
		ns, err := m.List()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return NotesLoadedMsg{Notes: ns}
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (e Explorer) Update(msg tea.Msg) (Explorer, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		e.list.SetSize(msg.Width, msg.Height-1)

	case NotesLoadedMsg:
		items := make([]list.Item, len(msg.Notes))
		for i, n := range msg.Notes {
			items[i] = noteItem{note: n}
		}
		e.list.SetItems(items)

	case NoteCreatedMsg:
		return e, tea.Batch(
			loadNotesCmd(e.manager),
			func() tea.Msg { return OpenNoteMsg{Note: msg.Note} },
		)

	case ErrMsg:
		e.err = msg.Err

	case tea.KeyMsg:
		if e.state == stateCreating {
			return e.updateCreating(msg)
		}
		if e.state == stateConfirmDelete {
			return e.updateConfirmDelete(msg)
		}
		return e.updateList(msg)
	}

	var cmd tea.Cmd
	e.list, cmd = e.list.Update(msg)
	return e, cmd
}

func (e Explorer) updateList(msg tea.KeyMsg) (Explorer, tea.Cmd) {
	switch msg.String() {

	case "enter":
		if item, ok := e.list.SelectedItem().(noteItem); ok {
			return e, func() tea.Msg { return OpenNoteMsg{Note: item.note} }
		}

	case "n":
		e.state = stateCreating
		e.input.SetValue("")
		e.input.Focus()
		return e, textinput.Blink

	case "d":
		if _, ok := e.list.SelectedItem().(noteItem); ok {
			e.state = stateConfirmDelete
		}
		return e, nil

	case "q":
		return e, tea.Quit
	}

	var cmd tea.Cmd
	e.list, cmd = e.list.Update(msg)
	return e, cmd
}

func (e Explorer) updateCreating(msg tea.KeyMsg) (Explorer, tea.Cmd) {
	switch msg.String() {

	case "enter":
		name := strings.TrimSpace(e.input.Value())
		if name == "" {
			e.state = stateList
			return e, nil
		}
		e.state = stateList
		return e, func() tea.Msg {
			nota, err := e.manager.Create(name)
			if err != nil {
				return ErrMsg{Err: err}
			}
			return NoteCreatedMsg{Note: nota}
		}

	case "esc":
		e.state = stateList
		e.input.Blur()
		return e, nil
	}

	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return e, cmd
}

func (e Explorer) updateConfirmDelete(msg tea.KeyMsg) (Explorer, tea.Cmd) {
	switch msg.String() {

	case "y", "Y":
		if item, ok := e.list.SelectedItem().(noteItem); ok {
			e.state = stateList
			return e, func() tea.Msg {
				if err := e.manager.Delete(item.note); err != nil {
					return ErrMsg{Err: err}
				}
				return loadNotesCmd(e.manager)()
			}
		}

	case "n", "N", "esc":
		e.state = stateList
	}
	return e, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (e Explorer) View() string {
	if e.state == stateCreating {
		return e.viewCreating()
	}
	if e.state == stateConfirmDelete {
		return e.viewConfirmDelete()
	}
	return e.viewList()
}

func (e Explorer) viewList() string {
	var b strings.Builder
	b.WriteString(e.list.View())

	// Tu statusbar personalizado
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		PaddingLeft(1)

	keys := []string{"↑↓ navegar", "Enter abrir", "n nueva", "d borrar", "/ buscar", "q salir"}
	b.WriteString("\n" + hintStyle.Render(strings.Join(keys, "  ·  ")))

	if e.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		b.WriteString("\n" + errStyle.Render("error: "+e.err.Error()))
	}

	return b.String()
}

func (e Explorer) viewCreating() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2).
		Width(40)

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Nueva nota"),
		"",
		e.input.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Enter para crear · Esc para cancelar"),
	)

	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, style.Render(content))
}

func (e Explorer) viewConfirmDelete() string {
	item, ok := e.list.SelectedItem().(noteItem)
	if !ok {
		return e.viewList()
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(1, 2).
		Width(40)

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")).Render("¿Eliminar nota?"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render(item.note.Name),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("y confirmar · n/Esc cancelar"),
	)

	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, style.Render(content))
}
