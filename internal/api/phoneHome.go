package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/lldp"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/google/gopacket/pcap"
	"go.uber.org/zap"
	"os"
	"strings"
	"sync"
	"time"
)

// ConstantlyPhoneHome sends every minute a single phone-home
// provisioning event to metal-api for each machine that sent at least one
// phone-home LLDP package to any interface of the host machine
// during this interval.
func (c *apiClient) ConstantlyPhoneHome() {
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		zapup.MustRootLogger().Error("unable to find interfaces",
			zap.Error(err),
		)
		os.Exit(1)
	}

	frameFragmentChan := make(chan lldp.FrameFragment)
	m := make(map[string]*lldp.PhoneHomeMessage)
	mtx := sync.Mutex{}
	e := event.NewEmitter(c.AppContext)

	for _, iface := range ifs {
		// consider only switch port interfaces
		if !strings.HasPrefix(iface.Name, "swp") {
			continue
		}
		lldpcli, err := lldp.NewClient(iface.Name)
		if err != nil {
			zapup.MustRootLogger().Error("unable to start LLDP client",
				zap.String("interface", iface.Name),
				zap.Error(err),
			)
			continue
		}

		// constantly observe LLDP traffic on current machine on current interface
		go lldpcli.CatchPackages(frameFragmentChan)

		// extract phone-home messages from fetched LLDP packages after a short initial delay
		time.AfterFunc(50*time.Second, func() {
			for phoneHome := range frameFragmentChan {
				msg := lldpcli.ExtractPhoneHomeMessage(&phoneHome)
				if msg == nil {
					continue
				}

				mtx.Lock()
				_, ok := m[msg.MachineID]
				if !ok {
					m[msg.MachineID] = msg
					// send first incoming message per machine immediately
					e.SendPhoneHomeEvent(msg)
				}
				mtx.Unlock()
			}
		})
	}

	// send a provisioning event to metal-api every minute for each reported-back machine
	t := time.NewTicker(1 * time.Minute)
	go func() {
		for range t.C {
			mtx.Lock()
			for machineID, msg := range m {
				e.SendPhoneHomeEvent(msg)
				delete(m, machineID)
			}
			mtx.Unlock()
		}
	}()
}
