//go:build client
// +build client

package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
)

const (
	provisioningEventPhonedHome = "Phoned Home"
)

// LLDPInterface represents the parsed LLDP JSON structure
type LLDPInterface struct {
	Via     string `json:"via"`
	Chassis map[string]struct {
		ID struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"id"`
		Descr string `json:"descr"`
	} `json:"chassis"`
}

// ConstantlyPhoneHome sends every minute a single phone-home
// provisioning event to metal-api for each machine that sent at least one
// phone-home LLDP package to any interface of the host machine
// during this interval.
func (c *Core) ConstantlyPhoneHome(ctx context.Context, interval time.Duration) {
	ifs, err := c.nos.GetSwitchPorts()
	if err != nil {
		c.log.Error("unable to find interfaces", "error", err)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult, 100)
	var phoneHomeMessages sync.Map

	// Start polling lldpd instead of raw packet capture
	go c.pollLLDPD(ctx, discoveryResultChan)

	// extract phone home messages from fetched LLDP packages
	go func() {
		for phoneHome := range discoveryResultChan {
			msg := toPhoneHomeMessage(phoneHome)
			if msg == nil {
				continue
			}

			phoneHomeMessages.Store(msg.machineID, *msg)
		}
	}()

	// send arrived messages on a ticker basis
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
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
		case <-ctx.Done():
			close(discoveryResultChan)
			return
		}
	}
}

// pollLLDPD queries lldpd periodically via docker exec
func (c *Core) pollLLDPD(ctx context.Context, resultChan chan<- lldp.DiscoveryResult) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "docker", "exec", "lldp", "lldpcli", "show", "neighbors", "-f", "json")
			output, err := cmd.Output()
			if err != nil {
				c.log.Error("failed to query lldpd", "error", err)
				continue
			}

			// Parse the JSON structure
			var result struct {
				Lldp struct {
					Interface []map[string]LLDPInterface `json:"interface"`
				} `json:"lldp"`
			}

			if err := json.Unmarshal(output, &result); err != nil {
				c.log.Error("failed to parse lldpd output", "error", err)
				continue
			}

			// Extract neighbors from each interface
			for _, ifaceMap := range result.Lldp.Interface {
				for ifaceName, ifaceData := range ifaceMap {
					// Each interface can have chassis info
					for chassisName, chassisData := range ifaceData.Chassis {
						discoveryResult := lldp.DiscoveryResult{
							InterfaceName:  ifaceName,
							SysName:        chassisName,
							SysDescription: chassisData.Descr,
						}
						resultChan <- discoveryResult
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *Core) send(ctx context.Context, event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	s, err := c.eventServiceClient.Send(ctx, event)
	if err != nil {
		return nil, err
	}
	if s != nil {
		c.log.Info("event", "send", s.GetEvents(), "failed", s.GetFailed())
	}
	return s, err
}

func (c *Core) phoneHome(ctx context.Context, msgs []phoneHomeMessage) {
	c.log.Info("phonehome", "machines", len(msgs))

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
		c.log.Error("unable to send provisioning event back to API", "error", err)
		c.metrics.CountError("send-provisioning")
	}
	if s != nil {
		c.log.Info("phonehome sent", "machines", s.GetEvents())
	}
}

// phoneHomeMessage contains a phone-home message.
type phoneHomeMessage struct {
	machineID string
	payload   string
	time      time.Time
}

// toPhoneHomeMessage extracts the machineID and payload of the given lldp frame fragment.
// Now accepts both "provisioned" and "metal-hammer" (waiting for installation) messages.
func toPhoneHomeMessage(discoveryResult lldp.DiscoveryResult) *phoneHomeMessage {
	descr := discoveryResult.SysDescription

	// Accept both provisioned machines and machines waiting for installation
	if strings.Contains(descr, "provisioned") || strings.Contains(descr, "metal-hammer") {
		return &phoneHomeMessage{
			machineID: discoveryResult.SysName,
			payload:   descr,
			time:      time.Now(),
		}
	}
	return nil
}
