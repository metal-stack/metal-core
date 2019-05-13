package switcher

import (
	"html/template"
	"io"
)

// Applier is an interface for rendering and reloading a switch configuration change
type Applier interface {
	Render(w io.Writer) error
	Reload() error
}

func render(t string, d interface{}, w io.Writer) error {
	tmpl, err := template.New(t).Parse(t)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, d)
}
