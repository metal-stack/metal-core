package core

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"net/http"
)

func NewBootService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"boot"}

	ws.Route(ws.GET("/v1/boot/{mac}").
		To(bootEndpoint).
		Doc("boot device by mac").
		Param(ws.PathParameter("mac", "mac of a network interface of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(BootResponse{}).
		Returns(http.StatusOK, "OK", BootResponse{}).
		Returns(http.StatusAccepted, "Accepted", BootResponse{}))

	return ws
}
