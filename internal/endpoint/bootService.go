package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/httperrors"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"net/http"
)

func (h *endpointHandler) NewBootService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"boot"}

	ws.Route(ws.GET("/v1/boot/{mac}").
		To(h.Boot).
		Doc("boot machine by mac").
		Param(ws.PathParameter("mac", "mac of a network interface of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(domain.BootResponse{}).
		Returns(http.StatusOK, "OK", domain.BootResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	ws.Route(ws.GET("/v1/dhcp/{id}").
		To(h.Dhcp).
		Doc("extended dhcp pxeboot request from a machine with guid").
		Param(ws.PathParameter("id", "the guid of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(domain.BootResponse{}).
		Returns(http.StatusOK, "OK", domain.BootResponse{}).
		DefaultReturns("Error", httperrors.HTTPErrorResponse{}))

	return ws
}
