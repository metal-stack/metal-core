package endpoint

import (
	"github.com/metal-stack/metal-core/pkg/domain"
)

type endpointHandler struct {
	*domain.AppContext
}

func NewHandler(ctx *domain.AppContext) domain.EndpointHandler {
	return &endpointHandler{ctx}
}
