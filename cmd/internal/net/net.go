package net

import (
	"fmt"
	"net"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
)

func GetLinkStatus(nicname string) (apiv2.SwitchPortStatus, error) {
	nic, err := net.InterfaceByName(nicname)
	if err != nil {
		return apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UNKNOWN, fmt.Errorf("cannot query interface %q : %w", nicname, err)
	}
	isup := nic.Flags&net.FlagRunning != 0
	if isup {
		return apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP, nil
	}
	return apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN, nil
}
