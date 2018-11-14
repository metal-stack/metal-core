package domain

import (
	"errors"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

type (
	BootResponse struct {
		Kernel      string   `json:"kernel,omitempty"`
		InitRamDisk []string `json:"initrd"`
		CommandLine string   `json:"cmdline,omitempty"`
	}

	// Report is send back to metal-core after installation finished
	Report struct {
		Success bool   `json:"success,omitempty" description:"true if installation succeeded"`
		Message string `json:"message" description:"if installation failed, the error message"`
	}

	MetalHammerRegisterDeviceRequest struct {
		models.MetalDeviceHardware
		UUID string            `json:"uuid,omitempty" description:"the uuid of the device to register"`
		IPMI *models.MetalIPMI `json:"ipmi" description:"the IPMI connection configuration"`
	}

	RestResponse struct {
		*restful.Response
	}
)

func (r *RestResponse) RespondError(statusCode int, errMsg string) {
	err := r.WriteError(statusCode, errors.New(errMsg))
	if err != nil {
		zapup.MustRootLogger().Error(err.Error())
		return
	}

	r.Flush()

	zapup.MustRootLogger().Error("Sent error response",
		zap.Int("statusCode", statusCode),
		zap.String("error", errMsg),
		zap.Error(err),
	)
}

func (r *RestResponse) Respond(statusCode int, body interface{}) {
	if body == nil {
		zapup.MustRootLogger().Info("Sent empty response",
			zap.Int("statusCode", statusCode),
		)
		return
	}

	err := r.WriteEntity(body)
	if err != nil {
		zapup.MustRootLogger().Error("Cannot write body",
			zap.Any("body", body),
			zap.Error(err),
		)
		return
	}

	r.Flush()

	zapup.MustRootLogger().Info("Sent response",
		zap.Int("statusCode", statusCode),
		zap.Any("body", body),
	)
}
