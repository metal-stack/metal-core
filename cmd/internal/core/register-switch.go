package core

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func (c *Core) RegisterSwitch() error {
	c.log.Infow("register switch")
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
		c.log.Errorw("unable to register at metal-api, retrying", "error", err)
		time.Sleep(30 * time.Second)
	}
	c.log.Infow("register switch completed", "params", params)
	return nil
}

func getNics(log *zap.SugaredLogger, blacklist []string) ([]*models.V1SwitchNic, error) {
	var nics []*models.V1SwitchNic
	links, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get all links: %w", err)
	}
	for _, l := range links {
		name := l.Name
		mac := l.HardwareAddr.String()

		if slices.Contains(blacklist, name) {
			log.Debugw("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}
		if !strings.HasPrefix(name, "swp") {
			log.Debugw("skip interface, because only swp* switch ports are reported to metal-api", "interface", name, "MAC", mac)
			continue
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
