package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) InstallImage(deviceId string) (int, *domain.Device) {
	dev := &domain.Device{}
	if sc := c.getExpect(fmt.Sprintf("/device/%v/wait", deviceId), nil, dev); sc != http.StatusOK {
		logging.Decorate(log.WithFields(log.Fields{
			"deviceID":   deviceId,
			"statusCode": sc,
		})).Error("Failed to GET installation image from Metal-APIs wait endpoint")
		return sc, nil
	} else {
		return http.StatusOK, dev
	}
}
