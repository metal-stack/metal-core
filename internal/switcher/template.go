package switcher

import (
	"embed"
	"log"
	"path"
)

//go:embed tpl
var templates embed.FS

var interfacesTPL = mustReadTpl("interfaces.tpl")
var frrTPL = mustReadTpl("frr.tpl")

func mustReadTpl(tplName string) string {
	contents, err := templates.ReadFile(path.Join("tpl", tplName))
	if err != nil {
		log.Panic(err)
	}
	return string(contents)
}
