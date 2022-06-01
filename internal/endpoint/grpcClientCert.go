package endpoint

import (
	"net/http"
	"os"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

type grpcConfig struct {
	address        string
	caCertFile     string
	clientCertFile string
	clientKeyFile  string
}

func (h *endpointHandler) GrpcClientCert(request *restful.Request, response *restful.Response) {
	bb, err := os.ReadFile(h.grpcConfig.caCertFile)
	if err != nil {
		h.log.Error("failed to read gRPC CA cert",
			zap.String("file", h.grpcConfig.caCertFile),
			zap.Error(err),
		)
		rest.RespondError(h.log, response, http.StatusInternalServerError, "failed to read gRPC CA cert")
		return
	}
	caCert := string(bb)

	bb, err = os.ReadFile(h.grpcConfig.clientCertFile)
	if err != nil {
		h.log.Error("failed to read gRPC client cert",
			zap.String("file", h.grpcConfig.clientCertFile),
			zap.Error(err),
		)
		rest.RespondError(h.log, response, http.StatusInternalServerError, "failed to read gRPC client cert")
		return
	}
	clientCert := string(bb)

	bb, err = os.ReadFile(h.grpcConfig.clientKeyFile)
	if err != nil {
		h.log.Error("failed to read gRPC client key",
			zap.String("file", h.grpcConfig.clientKeyFile),
			zap.Error(err),
		)
		rest.RespondError(h.log, response, http.StatusInternalServerError, "failed to read gRPC client key")
		return
	}
	clientKey := string(bb)

	rest.Respond(h.log, response, http.StatusOK, domain.GrpcResponse{
		Address: h.grpcConfig.address,
		CACert:  caCert,
		Cert:    clientCert,
		Key:     clientKey,
	})
}
