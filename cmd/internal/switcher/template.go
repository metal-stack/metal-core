package switcher

import (
	"embed"
	"io"
	"path"
	"text/template"
)

//go:embed tpl
var templates embed.FS

func mustParseFile(path string) *template.Template {
	return template.Must(template.ParseFiles(path))
}

func mustParseFS(name string) *template.Template {
	return template.Must(template.ParseFS(templates, path.Join("tpl", name)))
}

type templateRenderer struct {
	tpl *template.Template
}

func (r *templateRenderer) Render(w io.Writer, c *Conf) error {
	return r.tpl.Execute(w, c)
}
