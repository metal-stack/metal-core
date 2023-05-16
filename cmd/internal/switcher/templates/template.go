package templates

import (
	"embed"
	"path"
	"text/template"
)

//go:embed tpl
var templates embed.FS

const (
	interfaceTpl  = "interfaces.tpl"
	cumulusFrrTpl = "cumulus_frr.tpl"
	sonicFrrTpl   = "sonic_frr.tpl"
)

func mustParseFile(path string) *template.Template {
	return template.Must(template.ParseFiles(path))
}

func mustParseFS(name string) *template.Template {
	return template.Must(template.ParseFS(templates, path.Join("tpl", name)))
}

func mustParse(fsTpl, customTplPath string) *template.Template {
	if customTplPath != "" {
		return mustParseFile(customTplPath)
	}
	return mustParseFS(fsTpl)
}

func InterfacesTemplate(customTplPath string) *template.Template {
	return mustParse(interfaceTpl, customTplPath)
}

func CumulusFrrTemplate(customTplPath string) *template.Template {
	return mustParse(cumulusFrrTpl, customTplPath)
}

func SonicFrrTemplate(customTplPath string) *template.Template {
	return mustParse(sonicFrrTpl, customTplPath)
}
