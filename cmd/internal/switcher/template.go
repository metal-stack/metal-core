package switcher

import (
	"embed"
	"path"
)

//go:embed tpl
var templates embed.FS

var interfacesTPL = mustReadTpl("interfaces.tpl")
var frrTPL = mustReadTpl("frr.tpl")

func mustReadTpl(tplName string) string {
	contents, err := templates.ReadFile(path.Join("tpl", tplName))
	if err != nil {
		panic(err)
	}
	return string(contents)
}
