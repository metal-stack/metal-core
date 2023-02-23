package core

import (
	"fmt"
	"net"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
)

func (c *Core) RegisterSwitch() error {
	c.log.Infow("register switch")
	var (
		err      error
		nics     []*models.V1SwitchNic
		hostname string
	)

	if nics, err = getNics(c.log, c.nos, c.additionalBridgePorts); err != nil {
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
		c.log.Errorw("unable to register at metal-api, retrying", "error", err)
		time.Sleep(30 * time.Second)
	}
	c.log.Infow("register switch completed")
	return nil
}

func getNics(log *zap.SugaredLogger, nos switcher.NOS, blacklist []string) ([]*models.V1SwitchNic, error) {
	var nics []*models.V1SwitchNic
	ifs, err := nos.GetSwitchPorts()
	if err != nil {
		return nil, fmt.Errorf("unable to get all ifs: %w", err)
	}
links:
	for _, iface := range ifs {
		name := iface.Name
		mac := iface.HardwareAddr.String()
		for _, b := range blacklist {
			if b == name {
				log.Debugw("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
				continue links
			}
		}
		_, err := net.ParseMAC(mac)
		if err != nil {
			log.Debugw("skip interface with invalid mac", "interface", name, "MAC", mac)
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
