package log

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"os"
)

var log *zap.Logger

func InitConsoleEncoder() {
	os.Setenv(zapup.KeyLogEncoding, "console")
}

func Get() *zap.Logger {
	if log == nil {
		os.Setenv(zapup.KeyFieldApp, "Metal-Core")
		log = zapup.MustRootLogger()
	}
	return log
}
