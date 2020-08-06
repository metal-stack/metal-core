package endpoint

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-core/pkg/domain"
	"net/http"
)

func (h *endpointHandler) NewCertsService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/certs").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"certs"}

	ws.Route(ws.GET("/grpc-client-cert").
		To(h.GrpcClientCert).
		Doc("retrieves the client certificate of the gRPC server").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(domain.GrpcResponse{}).
		Returns(http.StatusOK, "OK", domain.GrpcResponse{}).
		Returns(http.StatusInternalServerError, "Error", nil).
		DefaultReturns("Error", nil))

	return ws
}
