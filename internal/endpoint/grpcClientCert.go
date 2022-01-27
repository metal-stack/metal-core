package endpoint

import (
	"net/http"
	"os"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

func (h *endpointHandler) GrpcClientCert(request *restful.Request, response *restful.Response) {
	bb, err := os.ReadFile(h.GrpcCACertFile)
	if err != nil {
		h.Log.Error("failed to read gRPC CA cert",
			zap.String("file", h.GrpcCACertFile),
			zap.Error(err),
		)
		rest.RespondError(h.Log, response, http.StatusInternalServerError, "failed to read gRPC CA cert")
		return
	}
	caCert := string(bb)

	bb, err = os.ReadFile(h.GrpcClientCertFile)
	if err != nil {
		h.Log.Error("failed to read gRPC client cert",
			zap.String("file", h.GrpcClientCertFile),
			zap.Error(err),
		)
		rest.RespondError(h.Log, response, http.StatusInternalServerError, "failed to read gRPC client cert")
		return
	}
	clientCert := string(bb)

	bb, err = os.ReadFile(h.GrpcClientKeyFile)
	if err != nil {
		h.Log.Error("failed to read gRPC client key",
			zap.String("file", h.GrpcClientKeyFile),
			zap.Error(err),
		)
		rest.RespondError(h.Log, response, http.StatusInternalServerError, "failed to read gRPC client key")
		return
	}
	clientKey := string(bb)

	rest.Respond(h.Log, response, http.StatusOK, domain.GrpcResponse{
		Address: h.GrpcAddress,
		CACert:  caCert,
		Cert:    clientCert,
		Key:     clientKey,
	})
}
