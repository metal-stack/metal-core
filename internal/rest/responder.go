package rest

import (
	"errors"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func RespondError(response *restful.Response, statusCode int, errMsg string) {
	//nolint:errcheck
	response.WriteError(statusCode, errors.New(errMsg))
	response.Flush()

	zapup.MustRootLogger().Error("Sent error response",
		zap.Int("statusCode", statusCode),
		zap.Error(errors.New(errMsg)),
	)
}

func Respond(response *restful.Response, statusCode int, body interface{}) {
	//nolint:errcheck
	response.WriteHeaderAndEntity(statusCode, body)
	response.Flush()

	zapup.MustRootLogger().Debug("Sent response",
		zap.Int("statusCode", statusCode),
		zap.Any("body", body),
	)
}
