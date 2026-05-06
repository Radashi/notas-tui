package notes

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manager struct {
	Dir string // Ruta de la carpeta
}

// Manager apuntando a la carpeta
// si no existe la crea

func NewManager(dir string) (*Manager, error) {
	expanded := expandHome(dir)

	if err := os.MkdirAll(expanded, 0o755); err != nil {
		return nil, err
	}

	return &Manager{Dir: expanded}, nil
}

// Lista que devuelve todas las notas ".md"
// ordenadas por fecha de modificacion
func (m *Manager) List() ([]Note, error) {
	entries, err := os.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}

	var notes []Note
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := entry.Name()
		notes = append(notes, Note{
			Name:     name,
			Title:    strings.TrimSuffix(name, ".md"),
			Path:     filepath.Join(m.Dir, name),
			Modified: info.ModTime(),
			Size:     info.Size(),
		})
	}

	// Esto pone las notas mas recientes primero
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Modified.After(notes[j].Modified)
	})

	return notes, nil
}

// Read devuelve el contenido de una notas
func (m *Manager) Read(note Note) (string, error) {
	data, err := os.ReadFile(note.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write guarda contenido en una nota, Crear archivo si no existe
func (m *Manager) Write(note Note, content string) error {
	return os.WriteFile(note.Path, []byte(content), 0o644)
}

// Create crea una nota nueva con un nombre dado
// Devuelve error si ya existe
func (m *Manager) Create(name string) (Note, error) {
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	// Verifica que no exista
	path := filepath.Join(m.Dir, name)
	if _, err := os.Stat(path); err == nil {
		return Note{}, os.ErrExist
	}

	// Crear header inicial
	initial := "# " + strings.TrimSuffix(name, ".md") + "\n\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		return Note{}, err
	}

	return Note{
		Name:     name,
		Title:    strings.TrimSuffix(name, ".md"),
		Path:     path,
		Modified: time.Now(),
	}, nil
}

// Delete elimina una nota permanentemente
func (m *Manager) Delete(note Note) error {
	return os.Remove(note.Path)
}

// Rename cambia el nombre de la nota
func (m *Manager) Rename(note Note, newName string) (Note, error) {
	if !strings.HasSuffix(newName, ".md") {
		newName = newName + ".md"
	}

	newPath := filepath.Join(m.Dir, newName)

	if err := os.Rename(note.Path, newPath); err != nil {
		return Note{}, err
	}

	return Note{
		Name:     newName,
		Title:    strings.TrimSuffix(newName, ".md"),
		Path:     newPath,
		Modified: time.Now(),
	}, nil
}

// expandHome remplaza ~ con el home directory real
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
