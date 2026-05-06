// cmd/main.go
package main

import (
	"fmt"
	"os"

	"github.com/Radashi/notas-tui/internal/notes"
	"github.com/Radashi/notas-tui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenExplorer screen = iota
	screenEditor
)

type app struct {
	screen   screen
	explorer ui.Explorer
	editor   ui.Editor
	manager  *notes.Manager
	width    int
	height   int
}

func (a app) Init() tea.Cmd {
	return a.explorer.Init()
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	// El explorador pide abrir una nota
	case ui.OpenNoteMsg:
		content, err := a.manager.Read(msg.Note)
		if err != nil {
			return a, nil
		}
		a.editor = ui.NewEditor(a.manager, msg.Note, content)
		a.screen = screenEditor
		// Mandar el tamaño actual al editor recién creado
		initCmd := func() tea.Msg {
			return tea.WindowSizeMsg{Width: a.width, Height: a.height}
		}
		return a, tea.Batch(a.editor.Init(), initCmd)
	// El editor pide cerrarse y volver
	case ui.CloseEditorMsg:
		a.screen = screenExplorer
		// Recargar la lista por si cambió algo
		return a, a.explorer.Init()

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		var cmd1 tea.Cmd
		a.explorer, cmd1 = a.explorer.Update(msg)
		if a.screen == screenEditor {
			var cmd2 tea.Cmd
			a.editor, cmd2 = a.editor.Update(msg)
			return a, tea.Batch(cmd1, cmd2)
		}
		return a, cmd1
	}

	// Delegar al modelo activo
	var cmd tea.Cmd
	switch a.screen {
	case screenExplorer:
		a.explorer, cmd = a.explorer.Update(msg)
	case screenEditor:
		a.editor, cmd = a.editor.Update(msg)
	}
	return a, cmd
}

func (a app) View() string {
	switch a.screen {
	case screenEditor:
		return a.editor.View()
	default:
		return a.explorer.View()
	}
}

func main() {
	manager, err := notes.NewManager("~/notas")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		app{
			screen:   screenExplorer,
			explorer: ui.NewExplorer(manager),
			manager:  manager,
		},
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
