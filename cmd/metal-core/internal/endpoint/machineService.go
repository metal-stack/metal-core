package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
)

func (h *endpointHandler) NewMachineService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/machine").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"machine"}

	ws.Route(ws.POST("/register/{id}").
		To(h.Register).
		Doc("register machine by ID").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.MetalHammerRegisterMachineRequest{}).
		Writes(models.MetalMachine{}).
		Returns(http.StatusOK, "OK", models.MetalMachine{}).
		Returns(http.StatusBadRequest, "Bad request", nil).
		Returns(http.StatusInternalServerError, "Accepted", nil))

	ws.Route(ws.GET("/install/{id}").
		To(h.Install).
		Doc("install machine by ID").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.MetalMachine{}).
		Returns(http.StatusOK, "OK", models.MetalMachineWithPhoneHomeToken{}).
		Returns(http.StatusNotModified, "No allocation", nil).
		Returns(http.StatusNotFound, "Not Found", nil))

	ws.Route(ws.POST("/report/{id}").
		To(h.Report).
		Doc("report machine by ID").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.Report{}).
		Writes(domain.BootResponse{}).
		Returns(http.StatusOK, "OK", models.MetalMachine{}).
		Returns(http.StatusNotAcceptable, "Not acceptable", nil).
		Returns(http.StatusInternalServerError, "Accepted", nil))

	return ws
}
