package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
)

func (e endpoint) Register(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerRegisterDeviceRequest{}

	err := request.ReadEntity(req)
	if err != nil {
		errMsg := "Unable to read body"
		zapup.MustRootLogger().Error("Cannot read request",
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusBadRequest, errMsg)
		return
	}

	devId := request.PathParameter("id")

	zapup.MustRootLogger().Info("Register device at Metal-API",
		zap.String("deviceID", devId),
		zap.String("IPMI-Address", impiAddress(req.IPMI)),
		zap.String("IPMI-Interface", impiInterface(req.IPMI)),
		zap.String("IPMI-MAC", impiMAC(req.IPMI)),
		zap.String("IPMI-User", impiUser(req.IPMI)),
	)

	sc, dev := e.ApiClient().RegisterDevice(devId, req)

	if sc != http.StatusOK {
		errMsg := "Failed to register device"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("deviceID", devId),
			zap.Any("device", dev),
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusInternalServerError, errMsg)
		return
	}

	zapup.MustRootLogger().Info("Device registered",
		zap.Int("statusCode", sc),
		zap.Any("device", dev),
	)
	rest.Respond(response, http.StatusOK, dev)
}

func impiAddress(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Address != nil {
		return *ipmi.Address
	}
	return ""
}

func impiInterface(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Interface != nil {
		return *ipmi.Interface
	}
	return ""
}

func impiMAC(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.Mac != nil {
		return *ipmi.Mac
	}
	return ""
}

func impiUser(ipmi *models.MetalIPMI) string {
	if ipmi != nil && ipmi.User != nil {
		return *ipmi.User
	}
	return ""
}
