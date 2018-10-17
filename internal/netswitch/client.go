package netswitch

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
)

type (
	Client interface {
		GetConfig() *domain.Config
	}
	client struct {
		Config *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	return client{
		Config: cfg,
	}
}

func (c client) GetConfig() *domain.Config {
	return c.Config
}
