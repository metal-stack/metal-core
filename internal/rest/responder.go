package rest

import (
	"errors"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

func RespondError(log *zap.Logger, response *restful.Response, statusCode int, errMsg string) {
	err := response.WriteErrorString(statusCode, errMsg)
	if err != nil {
		log.Error("Failed to send error response",
			zap.Any("designated error message", errMsg),
			zap.Error(err),
		)
		return
	}

	log.Error("Sent error response",
		zap.Int("statusCode", statusCode),
		zap.Error(errors.New(errMsg)),
	)
}

func Respond(log *zap.Logger, response *restful.Response, statusCode int, body interface{}) {
	err := response.WriteHeaderAndEntity(statusCode, body)
	if err != nil {
		log.Error("Failed to send response",
			zap.Any("designated body", body),
			zap.Error(err),
		)
		return
	}

	log.Debug("Sent response",
		zap.Int("statusCode", statusCode),
		zap.Any("body", body),
	)
}
