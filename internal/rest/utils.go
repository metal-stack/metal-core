package rest

import (
	"errors"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func RespondError(response *restful.Response, statusCode int, errMsg string) {
	if err := response.WriteError(statusCode, errors.New(errMsg)); err == nil {
		response.Flush()
		zapup.MustRootLogger().Error("Sent error response",
			zap.Int("statusCode", statusCode),
			zap.String("error", errMsg),
			zap.Error(err),
		)
	} else {
		zapup.MustRootLogger().Error(err.Error())
	}
}

func Respond(response *restful.Response, statusCode int, body interface{}) {
	if body == nil {
		zapup.MustRootLogger().Info("Sent empty response",
			zap.Int("statusCode", statusCode),
		)
	} else if err := response.WriteEntity(body); err != nil {
		zapup.MustRootLogger().Error("Cannot write body",
			zap.Any("body", body),
			zap.Error(err),
		)
	} else {
		response.Flush()
		zapup.MustRootLogger().Info("Sent response",
			zap.Int("statusCode", statusCode),
			zap.Any("body", body),
		)
	}
}
