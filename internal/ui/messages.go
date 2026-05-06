// internal/ui/messages.go
package ui

import "github.com/Radashi/notas-tui/internal/notes"

type (
	NotesLoadedMsg struct{ Notes []notes.Note }
	OpenNoteMsg    struct{ Note notes.Note }
	NoteCreatedMsg struct{ Note notes.Note }
	CloseEditorMsg struct{}
	OpenPreviewMsg struct{ Content string }
	ErrMsg         struct{ Err error }
)

func (e ErrMsg) Error() string { return e.Err.Error() }

// internas (solo las usa el editor)
type (
	noteSavedMsg    struct{}
	clearStatusMsg  struct{}
	closePreviewMsg struct{}
	ClosePreviewMsg struct{}
)
