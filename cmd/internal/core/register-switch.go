package core

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/avast/retry-go/v4"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/v"
)

func (c *Core) RegisterSwitch() error {
	c.log.Info("register switch")

	err := retry.Do(
		func() error {
			initialized, err := c.nos.IsInitialized()
			if err != nil {
				return err
			}
			if initialized {
				return nil
			}
			return fmt.Errorf("switch is not yet initialized")
		},
		retry.Attempts(120),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return fmt.Errorf("unable to register switch because it is not initialized: %w", err)
	}

	nics, err := c.nos.GetNics(c.log, c.additionalBridgePorts)
	if err != nil {
		return fmt.Errorf("unable to get nics: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("unable to get hostname: %w", err)
	}

	switchOS, err := c.nos.GetOS()
	if err != nil {
		return fmt.Errorf("unable to get switch os: %w", err)
	}
	switchOS.MetalCoreVersion = v.V.String()

	managementIP, managementUser, err := c.nos.GetManagement()
	if err != nil {
		return fmt.Errorf("unable to get switch management info: %w", err)
	}

	req := &infrav2.SwitchServiceRegisterRequest{
		Switch: &apiv2.Switch{
			Id:             hostname,
			Rack:           &c.rackID,
			Partition:      c.partitionID,
			ReplaceMode:    apiv2.SwitchReplaceMode_SWITCH_REPLACE_MODE_OPERATIONAL,
			ManagementIp:   managementIP,
			ManagementUser: pointer.Pointer(managementUser),
			Nics:           nics,
			Os:             switchOS,
		},
	}

	_ = retry.Do(
		func() error {
			if _, err := c.client.Infrav2().Switch().Register(context.TODO(), req); err == nil {
				return nil
			}
			c.log.Error("failed to register switch, retrying", "error", err)
			return err
		},
		retry.Attempts(0),
		retry.Delay(30*time.Second),
		retry.DelayType(retry.FixedDelay),
	)

	c.log.Info("register switch completed")
	return nil
}
