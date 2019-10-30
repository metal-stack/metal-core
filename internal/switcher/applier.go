package switcher

import (
	"io"
)

// Applier is an interface for rendering and reloading a switch configuration change
type Applier interface {
	Apply() error
	Render(w io.Writer) error
}
