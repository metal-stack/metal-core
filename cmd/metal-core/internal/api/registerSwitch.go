package api

import (
	"net"
	"os"
	"strings"
	"time"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func (c *apiClient) RegisterSwitch() (*models.V1SwitchResponse, error) {
	var err error
	var nics []*models.V1SwitchNic
	var hostname string

	if nics, err = getNics(c.AdditionalBridgePorts); err != nil {
		return nil, errors.Wrap(err, "unable to get nics")
	}

	if hostname, err = os.Hostname(); err != nil {
		return nil, errors.Wrap(err, "unable to get hostname")
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.V1SwitchRegisterRequest{
		ID:          &hostname,
		Name:        hostname,
		PartitionID: &c.PartitionID,
		RackID:      &c.RackID,
		Nics:        nics,
	}

	for {
		ok, created, err := c.SwitchClient.RegisterSwitch(params, c.Auth)
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

func getNics(blacklist []string) ([]*models.V1SwitchNic, error) {
	var nics []*models.V1SwitchNic
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
				continue links
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
		nic := &models.V1SwitchNic{
			Mac:  &mac,
			Name: &name,
		}
		nics = append(nics, nic)
	}
	return nics, nil
}
