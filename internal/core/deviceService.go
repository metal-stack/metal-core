package core

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

func NewDeviceService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/device").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"device"}

	ws.Route(ws.POST("/register/{id}").
		To(registerEndpoint).
		Doc("register device by ID").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.MetalHammerRegisterDeviceRequest{}).
		Writes(models.MetalDevice{}).
		Returns(http.StatusOK, "OK", models.MetalDevice{}).
		Returns(http.StatusBadRequest, "Bad request", nil).
		Returns(http.StatusInternalServerError, "Accepted", nil))

	ws.Route(ws.GET("/install/{id}").
		To(installEndpoint).
		Doc("install device by ID").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.MetalDevice{}).
		Returns(http.StatusOK, "OK", models.MetalDeviceWithPhoneHomeToken{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/report/{id}").
		To(reportEndpoint).
		Doc("report device by ID").
		Param(ws.PathParameter("id", "identifier of the device").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(Report{}).
		Writes(BootResponse{}).
		Returns(http.StatusOK, "OK", models.MetalDevice{}).
		Returns(http.StatusNotAcceptable, "Not acceptable", nil).
		Returns(http.StatusInternalServerError, "Accepted", nil))

	return ws
}
