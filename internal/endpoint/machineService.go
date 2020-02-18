package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/metal-stack/metal-core/models"
)

func (h *endpointHandler) NewMachineService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/machine").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"machine"}

	ws.Route(ws.GET("/{id}").
		To(h.FindMachine).
		Doc("find machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.V1MachineResponse{}).
		Returns(http.StatusOK, "OK", models.V1MachineResponse{}).
		Returns(http.StatusInternalServerError, "Error", nil))

	ws.Route(ws.POST("/register/{id}").
		To(h.Register).
		Doc("register machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.MetalHammerRegisterMachineRequest{}).
		Writes(models.V1MachineResponse{}).
		Returns(http.StatusOK, "OK", models.V1MachineResponse{}).
		Returns(http.StatusBadRequest, "Bad request", nil).
		Returns(http.StatusInternalServerError, "Error", nil))

	ws.Route(ws.GET("/install/{id}").
		To(h.Install).
		Doc("install machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.V1MachineResponse{}).
		Returns(http.StatusOK, "OK", models.V1MachineResponse{}).
		Returns(http.StatusNotModified, "No allocation", nil).
		Returns(http.StatusNotFound, "Not Found", nil).
		Returns(http.StatusExpectationFailed, "Incomplete machine", nil).
		Returns(http.StatusInternalServerError, "Error", nil))

	ws.Route(ws.POST("/report/{id}").
		To(h.Report).
		Doc("report machine").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.Report{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotAcceptable, "Not acceptable", nil).
		Returns(http.StatusInternalServerError, "Error", nil))

	ws.Route(ws.POST("/abort-reinstall/{id}").
		To(h.AbortReinstall).
		Doc("abort reinstall").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(domain.ChangeBootOrder{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusInternalServerError, "Error", nil))

	ws.Route(ws.POST("/{id}/event").
		To(h.AddProvisioningEvent).
		Doc("adds a machine provisioning event").
		Param(ws.PathParameter("id", "identifier of the machine").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(models.V1MachineProvisioningEvent{}).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Not Found", nil).
		DefaultReturns("Unexpected Error", nil))

	return ws
}
