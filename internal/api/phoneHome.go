package api

import (
	"github.com/google/gopacket/pcap"
	"github.com/metal-stack/metal-core/internal/lldp"
	"github.com/metal-stack/metal-lib/zapup"
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
	mtx := new(sync.RWMutex)

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

		// constantly observe LLDP traffic on current machine and current interface
		go lldpcli.CatchPackages(frameFragmentChan)

		// extract phone home messages from fetched LLDP packages after a short initial delay
		time.AfterFunc(50*time.Second, func() {
			for phoneHome := range frameFragmentChan {
				msg := lldpcli.ExtractPhoneHomeMessage(&phoneHome)
				if msg == nil {
					continue
				}

				mtx.RLock()
				_, ok := m[msg.MachineID]
				mtx.RUnlock()
				if !ok {
					mtx.Lock()
					m[msg.MachineID] = msg
					mtx.Unlock()
					// send first incoming phone home message per machine immediately
					c.PhoneHome(msg)
				}
			}
		})
	}

	// send phone home messages for each reported-back machine to metal-api every minute
	t := time.NewTicker(1 * time.Minute)
	go func() {
		for range t.C {
			// buffer phone home messages from map and clear it
			mtx.Lock()
			var mm []*lldp.PhoneHomeMessage
			for machineID, msg := range m {
				mm = append(mm, msg)
				delete(m, machineID)
			}
			mtx.Unlock()

			// send buffered phone home messages
			for _, msg := range mm {
				c.PhoneHome(msg)
			}
		}
	}()
}
