// internal/ui/editor.go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Radashi/notas-tui/internal/notes"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Estado interno ────────────────────────────────────────────────────────────

type editorState int

const (
	stateEditing     editorState = iota // editando normal
	stateSearching                      // Ctrl+F, buscando texto
	stateSaveConfirm                    // cambios sin guardar al salir
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Editor struct {
	note         notes.Note
	manager      *notes.Manager
	textarea     textarea.Model
	searchInput  textinput.Model
	state        editorState
	savedContent string // último contenido guardado (para detectar cambios)
	statusMsg    string // mensaje temporal en el statusbar ("guardado ✓")
	searchQuery  string
	width        int
	height       int
}

func NewEditor(manager *notes.Manager, note notes.Note, content string) Editor {
	// ── Textarea ──
	ta := textarea.New()
	ta.SetValue(content)
	ta.Focus()

	// Sin estilos de borde propios — nosotros controlamos el layout
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()

	// Wrap de líneas
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.ShowLineNumbers = true

	// ── Search input ──
	si := textinput.New()
	si.Placeholder = "buscar..."
	si.CharLimit = 80

	return Editor{
		note:         note,
		manager:      manager,
		textarea:     ta,
		searchInput:  si,
		state:        stateEditing,
		savedContent: content,
	}
}

// Dirty reporta si hay cambios sin guardar
func (e Editor) Dirty() bool {
	return e.textarea.Value() != e.savedContent
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (e Editor) Init() tea.Cmd {
	return textarea.Blink
}

// ── Update ────────────────────────────────────────────────────────────────────

func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		e.textarea.SetWidth(msg.Width - 4)
		e.textarea.SetHeight(msg.Height - 6) // espacio para header y statusbar

	case noteSavedMsg:
		e.savedContent = e.textarea.Value()
		e.statusMsg = "guardado ✓"
		// Limpiar el mensaje después de 2 segundos
		return e, clearStatusCmd()

	case clearStatusMsg:
		e.statusMsg = ""

	case tea.KeyMsg:
		switch e.state {
		case stateSearching:
			return e.updateSearching(msg)
		case stateSaveConfirm:
			return e.updateSaveConfirm(msg)
		default:
			return e.updateEditing(msg)
		}
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

func (e Editor) updateEditing(msg tea.KeyMsg) (Editor, tea.Cmd) {
	switch msg.String() {

	case "ctrl+s":
		return e, saveNoteCmd(e.manager, e.note, e.textarea.Value())

	case "ctrl+f":
		e.state = stateSearching
		e.searchInput.SetValue("")
		e.searchInput.Focus()
		return e, textinput.Blink

	case "ctrl+p":
		// Pasar al preview
		return e, func() tea.Msg {
			return OpenPreviewMsg{Content: e.textarea.Value()}
		}

	case "ctrl+a":
		// Seleccionar todo — mover cursor al final es lo más cercano
		// que bubbles/textarea soporta por ahora
		e.textarea.CursorEnd()
		return e, nil

	case "esc":
		if e.Dirty() {
			e.state = stateSaveConfirm
			return e, nil
		}
		return e, func() tea.Msg { return CloseEditorMsg{} }
	}

	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return e, cmd
}

func (e Editor) updateSearching(msg tea.KeyMsg) (Editor, tea.Cmd) {
	switch msg.String() {

	case "enter":
		e.searchQuery = e.searchInput.Value()
		e.state = stateEditing
		e.searchInput.Blur()
		// El highlight de búsqueda lo hacemos en el View
		return e, nil

	case "esc":
		e.searchQuery = ""
		e.state = stateEditing
		e.searchInput.Blur()
		return e, nil
	}

	var cmd tea.Cmd
	e.searchInput, cmd = e.searchInput.Update(msg)
	return e, cmd
}

func (e Editor) updateSaveConfirm(msg tea.KeyMsg) (Editor, tea.Cmd) {
	switch msg.String() {

	case "s", "S":
		// Guardar y salir
		return e, tea.Batch(
			saveNoteCmd(e.manager, e.note, e.textarea.Value()),
			func() tea.Msg { return CloseEditorMsg{} },
		)

	case "d", "D":
		// Descartar cambios y salir
		return e, func() tea.Msg { return CloseEditorMsg{} }

	case "esc", "c", "C":
		// Cancelar, volver a editar
		e.state = stateEditing
		return e, nil
	}
	return e, nil
}

// ── Comandos ──────────────────────────────────────────────────────────────────

func saveNoteCmd(m *notes.Manager, note notes.Note, content string) tea.Cmd {
	return func() tea.Msg {
		if err := m.Write(note, content); err != nil {
			return ErrMsg{err}
		}
		return noteSavedMsg{}
	}
}

// clearStatusMsg se manda después del timer para limpiar el statusbar

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2e9, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// ── View ──────────────────────────────────────────────────────────────────────

func (e Editor) View() string {
	if e.state == stateSaveConfirm {
		return e.viewSaveConfirm()
	}
	return e.viewEditor()
}

func (e Editor) viewEditor() string {
	// ── Estilos ──
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		PaddingLeft(1)

	dirtyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // amarillo
		PaddingLeft(1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // verde
		PaddingLeft(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		PaddingLeft(1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Padding(0, 1)

	// ── Header ──
	dirty := ""
	if e.Dirty() {
		dirty = dirtyStyle.Render("●") // punto amarillo = cambios sin guardar
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Render(e.note.Title+".md"),
		dirty,
	)

	// ── Contenido del textarea ──
	// Si hay búsqueda activa, resaltar ocurrencias en el texto
	textareaView := e.textarea.View()
	if e.searchQuery != "" {
		textareaView = highlightSearch(textareaView, e.searchQuery)
	}

	// ── Statusbar / barra inferior ──
	var statusBar string
	if e.state == stateSearching {
		statusBar = lipgloss.JoinHorizontal(lipgloss.Top,
			hintStyle.Render("buscar:"),
			" ",
			e.searchInput.View(),
			hintStyle.Render("  Enter confirmar · Esc cancelar"),
		)
	} else if e.statusMsg != "" {
		statusBar = statusStyle.Render(e.statusMsg)
	} else {
		// Keybindings hint
		keys := []string{"^S guardar", "^F buscar", "^P preview", "^Z deshacer", "Esc volver"}
		statusBar = hintStyle.Render(strings.Join(keys, "  ·  "))
	}

	// ── Línea de info (posición del cursor) ──
	row, col := e.textarea.CursorDown, 0
	_ = row
	_ = col
	infoLine := hintStyle.Render(fmt.Sprintf("%d líneas", strings.Count(e.textarea.Value(), "\n")+1))

	// ── Ensamblar ──
	body := borderStyle.
		Width(e.width - 2).
		Height(e.height - 5).
		Render(textareaView)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		infoLine,
		statusBar,
	)
}

func (e Editor) viewSaveConfirm() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(1, 3).
		Width(44)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Cambios sin guardar"),
		"",
		keyStyle.Render("s")+" "+hintStyle.Render("guardar y salir"),
		keyStyle.Render("d")+" "+hintStyle.Render("descartar y salir"),
		keyStyle.Render("c")+" "+hintStyle.Render("cancelar, seguir editando"),
	)

	return lipgloss.Place(
		e.width, e.height,
		lipgloss.Center, lipgloss.Center,
		style.Render(content),
	)
}

// highlightSearch marca las ocurrencias del query en el texto renderizado.
// Es una aproximación simple — resalta a nivel de string display.
func highlightSearch(text, query string) string {
	if query == "" {
		return text
	}
	highlight := lipgloss.NewStyle().
		Background(lipgloss.Color("3")).
		Foreground(lipgloss.Color("0")).
		Render(query)
	return strings.ReplaceAll(text, query, highlight)
}
