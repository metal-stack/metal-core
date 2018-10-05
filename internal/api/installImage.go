package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
)

func (c client) InstallImage(deviceId string) (int, *domain.Image) {
	var img *domain.Image
	sc := c.getExpect(fmt.Sprintf("/image/%v", "2"), nil, img)
	return sc, img
}
