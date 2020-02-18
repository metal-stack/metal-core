package api

import (
	"github.com/metal-stack/metal-core/pkg/domain"
)

type apiClient struct {
	*domain.AppContext
}

func NewClient(ctx *domain.AppContext) domain.APIClient {
	return &apiClient{
		AppContext: ctx,
	}
}
