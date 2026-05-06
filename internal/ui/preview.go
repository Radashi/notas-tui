package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// -- Modelo
type Preview struct {
	viewport viewport.Model
	title    string
	ready    bool
	width    int
	height   int
}

func NewPreview(title, content string, width, height int) Preview {
	if width == 0 {
		width = 80
	}

	if height == 0 {
		height = 24
	}

	p := Preview{
		title:  title,
		width:  width,
		height: height,
	}

	rendered := renderMarkdown(content, width)

	vp := viewport.New(width, height-5)
	vp.SetContent(rendered)

	return p
}

// Pasa el contenido por glamour y lo devuelve renderizado
func renderMarkdown(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return content // fallback: texto plano
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

// -- Init --
func (p Preview) Init() tea.Cmd {
	return nil
}

// -- Update --
func (p Preview) Update(msg tea.Msg) (Preview, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.viewport.Width = msg.Width
		p.viewport.Height = msg.Height - 5

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+p", "esc":
			// Volver al editor
			return p, func() tea.Msg { return ClosePreviewMsg{} }

		case "q":
			return p, func() tea.Msg { return CloseEditorMsg{} }
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// -- View ---
func (p Preview) View() string {
	if !p.ready {
		return "cargando..."
	}

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).PaddingLeft(1)
	badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(2)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingLeft(1)
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("5")).Padding(0, 1)
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerStyle.Render(p.title), badgeStyle.Render("· preview"))
	body := borderStyle.Width(p.width - 2).Height(p.height - 5).Render(p.viewport.View())

	// Scroll percent
	scrollInfo := fmt.Sprintf("%d%%", int(p.viewport.ScrollPercent()*100))
	info := hintStyle.Render(scrollInfo)

	hints := hintStyle.Render("↑↓/PgUp/PgDn scroll  ·  ^P/Esc volver editor  ·  q explorador")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		info,
		hints,
	)
}
