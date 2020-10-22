package rest

import (
	"errors"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func RespondError(response *restful.Response, statusCode int, errMsg string) {
	err := response.WriteErrorString(statusCode, errMsg)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send error response",
			zap.Any("designated error message", errMsg),
			zap.Error(err),
		)
		return
	}

	zapup.MustRootLogger().Error("Sent error response",
		zap.Int("statusCode", statusCode),
		zap.Error(errors.New(errMsg)),
	)
}

func Respond(response *restful.Response, statusCode int, body interface{}) {
	err := response.WriteHeaderAndEntity(statusCode, body)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to send response",
			zap.Any("designated body", body),
			zap.Error(err),
		)
		return
	}

	zapup.MustRootLogger().Debug("Sent response",
		zap.Int("statusCode", statusCode),
		zap.Any("body", body),
	)
}
