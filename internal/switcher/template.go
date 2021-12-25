package switcher

import (
	"embed"
	"log"
	"path"
)

//go:embed tpl
var templates embed.FS

func mustReadTpl(tplName string) string {
	contents, err := templates.ReadFile(path.Join("tpl", tplName))
	if err != nil {
		log.Panic(err)
		return ""
	}
	return string(contents)
}
