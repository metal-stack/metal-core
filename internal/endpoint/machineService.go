package endpoint

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-go/api/models"
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

	return ws
}
