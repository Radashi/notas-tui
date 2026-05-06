package notes

import "time"

type Note struct {
	Name     string
	Title    string
	Path     string
	Modified time.Time
	Size     int64
}
