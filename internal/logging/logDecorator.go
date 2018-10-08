package logging

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

var workDir string

func init() {
	if wd, err := os.Getwd(); err == nil {
		workDir = wd
	} else {
		workDir = ""
	}
}

func Decorate(logger *log.Entry) *log.Entry {
	st := ""
	for i := 1; i <= 5; i++ {
		if pc, file, line, ok := runtime.Caller(i); ok {
			funcName := runtime.FuncForPC(pc).Name()
			file, funcName = sanitize(file, funcName)
			if len(st) > 0{
				st += "\n"
			}
			st += fmt.Sprintf("%v::%v#%d", file, funcName, line)
		}
	}
	if len(st) >  0 {
		return logger.WithField("stackTrace", st)
	} else {
		return logger
	}
}

func sanitize(file, funcName string) (string, string) {
	if strings.Contains(file, workDir) {
		index := strings.Index(file, "/internal")
		if index != -1 {
			file = file[index:]
		}
		file = file[len(workDir):]
		return file, funcName[strings.LastIndex(funcName, ".")+1:]
	} else {
		return file, funcName
	}
}
