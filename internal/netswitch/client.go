package netswitch

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
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

func NewClient(cfg domain.Config) Client {
	return client{
		Config: cfg,
	}
}

func (c client) GetConfig() domain.Config {
	return c.Config
}

func (c client) ConfigurePorts([]domain.SwitchPort) int {
	return http.StatusOK
}
