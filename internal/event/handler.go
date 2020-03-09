package event

import (
	"time"

	"github.com/metal-stack/metal-core/pkg/domain"
)

type eventHandler struct {
	*domain.AppContext
	sr chan switchReconfigureEvent
}
type switchReconfigureEvent struct {
	switchName string
	eventType  string
	occurence  time.Time
}

func newSwitchReconfigureEvent(switchName, eventType string) switchReconfigureEvent {
	return switchReconfigureEvent{
		switchName: switchName,
		eventType:  eventType,
		occurence:  time.Now(),
	}
}

func NewHandler(ctx *domain.AppContext) domain.EventHandler {
	return &eventHandler{ctx, make(chan switchReconfigureEvent, 50)}
}
