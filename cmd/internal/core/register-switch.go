package core

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

func (c *Core) RegisterSwitch() error {
	c.log.Sugar().Infow("register switch")
	var (
		err      error
		nics     []*models.V1SwitchNic
		hostname string
	)

	if nics, err = getNics(c.log, c.additionalBridgePorts); err != nil {
		return fmt.Errorf("unable to get nics: %w", err)
	}

	if hostname, err = os.Hostname(); err != nil {
		return fmt.Errorf("unable to get hostname: %w", err)
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.V1SwitchRegisterRequest{
		ID:          &hostname,
		Name:        hostname,
		PartitionID: &c.partitionID,
		RackID:      &c.rackID,
		Nics:        nics,
	}

	for {
		_, _, err := c.driver.SwitchOperations().RegisterSwitch(params, nil)
		if err == nil {
			break
		}
		c.log.Error("unable to register at metal-api, retrying", zap.Error(err))
		time.Sleep(30 * time.Second)
	}
	c.log.Sugar().Infow("register switch completed")
	return nil
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
