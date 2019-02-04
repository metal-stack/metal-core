package api

import "git.f-i-ts.de/cloud-native/metal/metal-core/domain"

type apiClient struct {
	*domain.AppContext
}

func NewClient(ctx *domain.AppContext) domain.APIClient {
	return &apiClient{ctx}
}
