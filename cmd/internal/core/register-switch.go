package core

import (
	"fmt"
	"os"
	"time"

	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
)

func (c *Core) RegisterSwitch() error {
	c.log.Infow("register switch")
	var (
		err            error
		nics           []*models.V1SwitchNic
		hostname       string
		switchOS       *models.V1SwitchOS
		managementIP   string
		managementUser string
	)

	if nics, err = c.nos.GetNics(c.log, c.additionalBridgePorts); err != nil {
		return fmt.Errorf("unable to get nics: %w", err)
	}

	if hostname, err = os.Hostname(); err != nil {
		return fmt.Errorf("unable to get hostname: %w", err)
	}

	if switchOS, err = c.nos.GetOS(); err != nil {
		return fmt.Errorf("unable to get switch os: %w", err)
	}

	if managementIP, managementUser, err = c.nos.GetManagement(); err != nil {
		return fmt.Errorf("unable to get switch management info: %w", err)
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.V1SwitchRegisterRequest{
		ID:             &hostname,
		Name:           hostname,
		PartitionID:    &c.partitionID,
		RackID:         &c.rackID,
		Nics:           nics,
		Os:             switchOS,
		ManagementIP:   managementIP,
		ManagementUser: managementUser,
	}

	// TODO could be done with retry-go
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
