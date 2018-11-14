package domain

import "git.f-i-ts.de/cloud-native/maas/metal-core/models"

type (
	EventType string

	DeviceEvent struct {
		Type EventType           `json:"type,omitempty"`
		Old  *models.MetalDevice `json:"old,omitempty"`
		New  *models.MetalDevice `json:"new,omitempty"`
	}
)

// Some EventType enums.
const (
	CREATE EventType = "create"
	UPDATE EventType = "update"
	DELETE EventType = "delete"
)
