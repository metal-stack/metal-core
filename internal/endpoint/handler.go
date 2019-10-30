package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
)

type endpointHandler struct {
	*domain.AppContext
}

func NewHandler(ctx *domain.AppContext) domain.EndpointHandler {
	return &endpointHandler{ctx}
}
