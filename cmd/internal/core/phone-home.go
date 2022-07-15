package core

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"go.uber.org/zap"
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
func (c *Core) ConstantlyPhoneHome(ctx context.Context) {
	ifs, err := net.Interfaces()
	if err != nil {
		c.log.Errorw("unable to find interfaces", "error", err)
		os.Exit(1)
	}

	discoveryResultChan := make(chan lldp.DiscoveryResult)

	phoneHomeMessages := sync.Map{}
	// initial interface discovery
	for _, iface := range ifs {
		c.startLLDPDiscovery(ctx, discoveryResultChan, iface.Name)
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
				phoneHomeMessages.Range(func(key, value any) bool {
					msg, ok := value.(phoneHomeMessage)
					if !ok {
						return true
					}
					phoneHomeMessages.Delete(key)
					msgs = append(msgs, msg)
					return true
				})
				c.phoneHome(msgs)
			}
		}
	}()

	ifaceTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ifaceTicker.C:
				ifs, err := net.Interfaces()
				if err != nil {
					c.log.Errorw("unable to gather interfaces, ignoring", "error", err)
					continue
				}
				interfaces := []string{}
				existing := []string{}
				c.interfaces.Range(func(key, value any) bool {
					existing = append(existing, key.(string))
					return true
				})
				for _, iface := range ifs {
					interfaces = append(interfaces, iface.Name)
				}
				newInterfaces := interfaces
				for _, i := range existing {
					index := slices.Index(existing, i)
					if index < 0 {
						continue
					}
					newInterfaces = slices.Delete(newInterfaces, index, index)
				}
				removedInterfaces := interfaces
				for _, i := range interfaces {
					index := slices.Index(existing, i)
					if index < 0 {
						removedInterfaces = slices.Delete(removedInterfaces, index, index)
					}
				}
				for _, i := range removedInterfaces {
					c.stopLLDPDiscovery(i)
				}
				for _, i := range newInterfaces {
					c.startLLDPDiscovery(ctx, discoveryResultChan, i)
				}
			}
		}
	}()

	<-ctx.Done()
}

func (c *Core) send(event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func (c *Core) phoneHome(msgs []phoneHomeMessage) {
	c.log.Debug("phonehome", zap.Any("machines", msgs))
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

	s, err := c.send(&v1.EventServiceSendRequest{Events: events})
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

func (c *Core) startLLDPDiscovery(ctx context.Context, discoveryResultChan chan lldp.DiscoveryResult, i string) {
	value, ok := c.interfaces.Load(i)
	if !ok {
		return
	}
	iface := value.(net.Interface)
	// consider only switch port interfaces
	if !strings.HasPrefix(iface.Name, "swp") {
		return
	}
	ifacectx, cancel := context.WithCancel(ctx)
	lldpcli, err := lldp.NewClient(ifacectx, iface)
	if err != nil {
		c.log.Errorw("unable to start LLDP client", "interface", iface.Name, "error", err)
		return
	}
	c.log.Infow("start lldp client", "interface", iface.Name)

	// constantly observe LLDP traffic on current machine and current interface
	go lldpcli.Start(discoveryResultChan)

	c.interfaces.Store(iface.Name, iface)
	c.interfaceCancelFuncs.Store(iface.Name, cancel)
}

func (c *Core) stopLLDPDiscovery(iface string) {
	value, ok := c.interfaceCancelFuncs.Load(iface)
	if !ok {
		return
	}
	f := value.(context.CancelFunc)
	f()
	c.interfaceCancelFuncs.Delete(iface)
	c.interfaces.Delete(iface)
}
