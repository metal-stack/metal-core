package api

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/metal-stack/metal-core/internal/lldp"
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
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		zapup.MustRootLogger().Error("unable to find interfaces",
			zap.Error(err),
		)
		os.Exit(1)
	}

	frameFragmentChan := make(chan lldp.FrameFragment)
	m := make(map[string]time.Time)
	mtx := new(sync.Mutex)

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
		go func() {
			time.Sleep(InitialPhonedHomeBackoff)

			for phoneHome := range frameFragmentChan {
				msg := lldpcli.ExtractPhoneHomeMessage(&phoneHome)
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
