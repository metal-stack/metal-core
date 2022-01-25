package api

import (
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

const (
	InitialPhonedHomeBackoff = 20 * time.Second
	PhonedHomeBackoff        = 58 * time.Second // lldpd sends messages every two seconds
)

// ConstantlyPhoneHome sends every minute a single phone-home
// provisioning event to metal-api for each machine that sent at least one
// phone-home LLDP package to any interface of the host machine
// during this interval.
func (c *apiClient) ConstantlyPhoneHome() {
	// FIXME this list of interfaces is only read on startup
	// if additional interfaces are configured, no new lldpd client is started and therefore no
	// phoned home events are sent for these interfaces.
	// Solution:
	// - either ensure metal-core is restarted on interfaces added/removed
	// - dynamically detect changes and stop/start goroutines for the lldpd client per interface
	ifs, err := net.Interfaces()
	if err != nil {
		zapup.MustRootLogger().Error("unable to find interfaces",
			zap.Error(err),
		)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult)
	m := make(map[string]time.Time)
	mtx := new(sync.Mutex)

	for _, iface := range ifs {
		// consider only switch port interfaces
		if !strings.HasPrefix(iface.Name, "swp") {
			continue
		}
		lldpcli, err := lldp.NewClient(iface)
		if err != nil {
			zapup.MustRootLogger().Error("unable to start LLDP client",
				zap.String("interface", iface.Name),
				zap.Error(err),
			)
			continue
		}

		// constantly observe LLDP traffic on current machine and current interface
		go lldpcli.Start(discoveryResultChan)

		// extract phone home messages from fetched LLDP packages after a short initial delay
		go func() {
			time.Sleep(InitialPhonedHomeBackoff)

			for phoneHome := range discoveryResultChan {
				phoneHome := phoneHome
				msg := toPhoneHomeMessage(phoneHome)
				if msg == nil {
					continue
				}

				sendToAPI := false

				mtx.Lock()
				lastSend, ok := m[msg.MachineID]
				if !ok || time.Since(lastSend) > PhonedHomeBackoff {
					sendToAPI = true
					m[msg.MachineID] = time.Now()
				}
				mtx.Unlock()

				if sendToAPI {
					go c.PhoneHome(msg)
				}
			}
		}()
	}
}

// PhoneHomeMessage contains a phone-home message.
type PhoneHomeMessage struct {
	MachineID string
	Payload   string
}

// toPhoneHomeMessage extracts the machineID and payload of the given lldp frame fragment.
// An error will be returned if the frame fragment does not contain a phone-home message.
func toPhoneHomeMessage(discoveryResult lldp.DiscoveryResult) *PhoneHomeMessage {
	if strings.Contains(discoveryResult.SysDescription, "provisioned") {
		return &PhoneHomeMessage{
			MachineID: discoveryResult.SysName,
			Payload:   discoveryResult.SysDescription,
		}
	}
	return nil
}
