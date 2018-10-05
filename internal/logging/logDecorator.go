package logging

import (
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

var workDir string

func init() {
	if wd, err := os.Getwd(); err == nil && len(wd) > 13 {
		workDir = wd[0 : len(wd)-13]
	} else {
		workDir = ""
	}
}

func Decorate(logger *log.Entry) *log.Entry {
	if pc, file, line, ok := runtime.Caller(1); ok {
		funcName := runtime.FuncForPC(pc).Name()
		file, funcName = sanitize(file, funcName)
		return logger.WithFields(log.Fields{
			"file": file,
			"line": line,
			"func": funcName,
		})
	} else {
		return logger
	}
}

func sanitize(file, funcName string) (string, string) {
	if strings.Contains(file, workDir) {
		file = file[len(workDir):]
		return file, funcName[strings.LastIndex(funcName, ".")+1:]
	} else {
		return file, funcName
	}
}
