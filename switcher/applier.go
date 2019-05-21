package switcher

import (
	"io"
)

// Applier is an interface for rendering and reloading a switch configuration change
type Applier interface {
	Render(w io.Writer) error
	Reload() error
	Validate() error
}
