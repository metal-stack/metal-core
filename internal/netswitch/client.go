package netswitch

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"net/http"
)

type (
	Client interface {
		GetConfig() domain.Config
		ConfigurePorts([]domain.SwitchPort) int
	}
	client struct {
		Config domain.Config
	}
)

func NewClient(config domain.Config) Client {
	return client{
		Config: config,
	}
}

func (c client) GetConfig() domain.Config {
	return c.Config
}

func (c client) ConfigurePorts([]domain.SwitchPort) int {
	return http.StatusOK
}
