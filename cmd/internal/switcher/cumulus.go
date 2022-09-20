package switcher

import (
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

type Cumulus struct {
	frrApplier        *FrrApplier
	interfacesApplier *InterfacesApplier
	log               *zap.SugaredLogger
}

func NewCumulus(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) *Cumulus {
	return &Cumulus{
		frrApplier:        NewFrrApplier(frrTplFile),
		interfacesApplier: NewInterfacesApplier(interfacesTplFile),
		log:               log,
	}
}

func (c *Cumulus) Apply(cfg *Conf) error {
	err := c.interfacesApplier.Apply(cfg)
	if err != nil {
		return err
	}

	err = c.frrApplier.Apply(cfg)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cumulus) GetSwitchPorts() ([]*net.Interface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	switchPorts := make([]*net.Interface, 0, len(ifs))
	for i := range ifs {
		iface := &ifs[i]
		if !strings.HasPrefix(iface.Name, "swp") {
			c.log.Debug("skip interface, because only swp* interface are front panels",
				zap.String("interface", iface.Name),
				zap.String("MAC", iface.HardwareAddr.String()),
			)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	return switchPorts, nil
}
