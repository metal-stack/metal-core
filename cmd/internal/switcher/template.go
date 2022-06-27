package switcher

import (
	"embed"
	"path"
	"text/template"
)

//go:embed tpl
var templates embed.FS

func parseFileOrFallback(path string, fallbackFS string) *template.Template {
	tpl, err := template.ParseFiles(path)
	if err != nil {
		return mustParseFS(fallbackFS)
	}
	return tpl
}

func mustParseFS(name string) *template.Template {
	return template.Must(template.ParseFS(templates, path.Join("tpl", name)))
}
