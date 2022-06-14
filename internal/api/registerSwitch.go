package api

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func (c *apiClient) RegisterSwitch() (*models.V1SwitchResponse, error) {
	var err error
	var nics []*models.V1SwitchNic
	var hostname string

	if nics, err = getNics(c.Log, c.AdditionalBridgePorts); err != nil {
		return nil, fmt.Errorf("unable to get nics: %w", err)
	}

	if hostname, err = os.Hostname(); err != nil {
		return nil, fmt.Errorf("unable to get hostname: %w", err)
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
		ok, created, err := c.Driver.SwitchOperations().RegisterSwitch(params, nil)
		if err == nil {
			if ok != nil {
				return ok.Payload, nil //nolint
			}
			return created.Payload, nil
		}
		c.Log.Error("unable to register at metal-api", zap.Error(err))
		time.Sleep(time.Second)
	}
}

func getNics(log *zap.Logger, blacklist []string) ([]*models.V1SwitchNic, error) {
	var nics []*models.V1SwitchNic
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("unable to get all links: %w", err)
	}
links:
	for _, l := range links {
		attrs := l.Attrs()
		name := attrs.Name
		mac := attrs.HardwareAddr.String()
		for _, b := range blacklist {
			if b == name {
				log.Debug("skip interface, because it is contained in the blacklist",
					zap.String("interface", name),
					zap.Any("blacklist", blacklist),
				)
				continue links
			}
		}
		if !strings.HasPrefix(name, "swp") {
			log.Debug("skip interface, because only swp* switch ports are reported to metal-api",
				zap.String("interface", name),
				zap.String("MAC", mac),
			)
			continue
		}
		_, err := net.ParseMAC(mac)
		if err != nil {
			log.Debug("skip interface with invalid mac",
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
