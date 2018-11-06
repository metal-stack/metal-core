package netswitch

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
)

type (
	Client interface {
		Config() *domain.Config
	}
	client struct {
		config *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	return client{
		config: cfg,
	}
}

func (c client) Config() *domain.Config {
	return c.config
}
