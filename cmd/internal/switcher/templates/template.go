package templates

import (
	"embed"
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
