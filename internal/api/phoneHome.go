package api

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	"go.uber.org/zap"
)

const (
	phonedHomeInterval = time.Minute // lldpd sends messages every two seconds
)

// ConstantlyPhoneHome sends every minute a single phone-home
// provisioning event to metal-api for each machine that sent at least one
// phone-home LLDP package to any interface of the host machine
// during this interval.
func (c *ApiClient) ConstantlyPhoneHome() {
	// FIXME this list of interfaces is only read on startup
	// if additional interfaces are configured, no new lldpd client is started and therefore no
	// phoned home events are sent for these interfaces.
	// Solution:
	// - either ensure metal-core is restarted on interfaces added/removed
	// - dynamically detect changes and stop/start goroutines for the lldpd client per interface
	ifs, err := net.Interfaces()
	if err != nil {
		c.log.Error("unable to find interfaces",
			zap.Error(err),
		)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult)

	// FIXME context should come from caller and canceled on shutdown
	ctx := context.Background()

	phoneHomeMessages := sync.Map{}
	for _, iface := range ifs {
		// consider only switch port interfaces
		if !strings.HasPrefix(iface.Name, "swp") {
			continue
		}
		lldpcli, err := lldp.NewClient(ctx, iface)
		if err != nil {
			c.log.Error("unable to start LLDP client",
				zap.String("interface", iface.Name),
				zap.Error(err),
			)
			continue
		}
		c.log.Info("start lldp client",
			zap.String("interface", iface.Name),
		)

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
				c.PhoneHome(msgs)
			}
		}
	}()
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
