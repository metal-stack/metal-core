package api

import (
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"net"
	"os"
	"strings"
	"time"
)

func (c *apiClient) RegisterSwitch() (*models.MetalSwitch, error) {
	return registerSwitch(c.Config, c.SwitchClient)
}

func registerSwitch(cfg *domain.Config, switchClient *sw.Client) (*models.MetalSwitch, error) {
	var err error
	var nics []*models.MetalNic
	var hostname string

	if nics, err = getNics(cfg.AdditionalBridgePorts); err != nil {
		return nil, errors.Wrap(err, "unable to get nics")
	}

	if hostname, err = os.Hostname(); err != nil {
		return nil, errors.Wrap(err, "unable to get hostname")
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.MetalRegisterSwitch{
		ID:          &hostname,
		PartitionID: &cfg.PartitionID,
		RackID:      &cfg.RackID,
		Nics:        nics,
	}

	for {
		ok, created, err := switchClient.RegisterSwitch(params)
		if err == nil {
			if ok != nil {
				return ok.Payload, nil
			}
			return created.Payload, nil
		}
		zapup.MustRootLogger().Error("unable to register at metal-api", zap.Error(err))
		time.Sleep(time.Second)
	}
}

func getNics(blacklist []string) ([]*models.MetalNic, error) {
	var nics []*models.MetalNic
	links, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get all links")
	}
links:
	for _, l := range links {
		attrs := l.Attrs()
		name := attrs.Name
		mac := attrs.HardwareAddr.String()
		for _, b := range blacklist {
			if b == name {
				zapup.MustRootLogger().Info("skip interface, because it is contained in the blacklist",
					zap.String("interface", name),
					zap.Any("blacklist", blacklist),
				)
				break links
			}
		}
		if !strings.HasPrefix(name, "swp") {
			zapup.MustRootLogger().Info("skip interface, because only swp* switch ports are reported to metal-api",
				zap.String("interface", name),
				zap.String("MAC", mac),
			)
			continue
		}
		_, err := net.ParseMAC(mac)
		if err != nil {
			zapup.MustRootLogger().Info("skip interface with invalid mac",
				zap.String("interface", name),
				zap.String("MAC", mac),
			)
			continue
		}
		nic := &models.MetalNic{
			Mac:  &mac,
			Name: &name,
		}
		nics = append(nics, nic)
	}
	return nics, nil
}
