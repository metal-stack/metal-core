package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"net/http"
)

func (e endpoint) NewBootService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"boot"}

	ws.Route(ws.GET("/v1/boot/{mac}").
		To(e.Boot).
		Doc("boot device by mac").
		Param(ws.PathParameter("mac", "mac of a network interface of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(domain.BootResponse{}).
		Returns(http.StatusOK, "OK", domain.BootResponse{}).
		Returns(http.StatusAccepted, "Accepted", domain.BootResponse{}).
		Returns(http.StatusBadRequest, "Bad request", domain.BootResponse{}))

	return ws
}
