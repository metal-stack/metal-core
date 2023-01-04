//go:build client
// +build client

package core

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	phonedHomeInterval          = time.Minute // lldpd sends messages every two seconds
	provisioningEventPhonedHome = "Phoned Home"
)

// ConstantlyPhoneHome sends every minute a single phone-home
// provisioning event to metal-api for each machine that sent at least one
// phone-home LLDP package to any interface of the host machine
// during this interval.
func (c *Core) ConstantlyPhoneHome() {
	// FIXME this list of interfaces is only read on startup
	// if additional interfaces are configured, no new lldpd client is started and therefore no
	// phoned home events are sent for these interfaces.
	// Solution:
	// - either ensure metal-core is restarted on interfaces added/removed
	// - dynamically detect changes and stop/start goroutines for the lldpd client per interface
	ifs, err := c.nos.GetSwitchPorts()
	if err != nil {
		c.log.Errorw("unable to find interfaces", "error", err)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult)

	// FIXME context should come from caller and canceled on shutdown
	ctx := context.Background()

	phoneHomeMessages := sync.Map{}
	for _, iface := range ifs {
		lldpcli, err := lldp.NewClient(ctx, *iface)
		if err != nil {
			c.log.Errorw("unable to start LLDP client", "interface", iface.Name, "error", err)
			continue
		}
		c.log.Infow("start lldp client", "interface", iface.Name)

		// constantly observe LLDP traffic on current machine and current interface
		go lldpcli.Start(discoveryResultChan)

	}
	// extract phone home messages from fetched LLDP packages
	go func() {
		for phoneHome := range discoveryResultChan {
			phoneHome := phoneHome
			msg := toPhoneHomeMessage(phoneHome)
			if msg == nil {
				continue
			}

			phoneHomeMessages.Store(msg.machineID, *msg)
		}
	}()

	// send arrived messages on a ticker basis
	ticker := time.NewTicker(phonedHomeInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msgs := []phoneHomeMessage{}
				phoneHomeMessages.Range(func(key, value interface{}) bool {
					msg, ok := value.(phoneHomeMessage)
					if !ok {
						return true
					}
					phoneHomeMessages.Delete(key)
					msgs = append(msgs, msg)
					return true
				})
				c.phoneHome(ctx, msgs)
			}
		}
	}()
}

func (c *Core) send(ctx context.Context, event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	s, err := c.eventServiceClient.Send(ctx, event)
	if err != nil {
		return nil, err
	}
	if s != nil {
		c.log.Infow("event", "send", s.Events, "failed", s.Failed)
	}
	return s, err
}

func (c *Core) phoneHome(ctx context.Context, msgs []phoneHomeMessage) {
	c.log.Debugw("phonehome", "machines", msgs)
	c.log.Infow("phonehome", "machines", len(msgs))

	events := make(map[string]*v1.MachineProvisioningEvent)
	phonedHomeEvent := string(provisioningEventPhonedHome)
	for i := range msgs {
		msg := msgs[i]
		event := &v1.MachineProvisioningEvent{
			Event:   phonedHomeEvent,
			Message: msg.payload,
			Time:    timestamppb.New(msg.time),
		}
		events[msg.machineID] = event
	}

	s, err := c.send(ctx, &v1.EventServiceSendRequest{Events: events})
	if err != nil {
		c.log.Errorw("unable to send provisioning event back to API", "error", err)
	}
	if s != nil {
		c.log.Infow("phonehome sent", "machines", s.Events)
	}
}

// phoneHomeMessage contains a phone-home message.
type phoneHomeMessage struct {
	machineID string
	payload   string
	time      time.Time
}

// toPhoneHomeMessage extracts the machineID and payload of the given lldp frame fragment.
// An error will be returned if the frame fragment does not contain a phone-home message.
func toPhoneHomeMessage(discoveryResult lldp.DiscoveryResult) *phoneHomeMessage {
	if strings.Contains(discoveryResult.SysDescription, "provisioned") {
		return &phoneHomeMessage{
			machineID: discoveryResult.SysName,
			payload:   discoveryResult.SysDescription,
			time:      time.Now(),
		}
	}
	return nil
}
