package cumulus

import (
	"fmt"
	"net"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"

	"go.uber.org/zap"
)

type Cumulus struct {
	frrApplier        *templates.FrrApplier
	interfacesApplier *templates.InterfacesApplier
	log               *zap.SugaredLogger
}

func New(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) *Cumulus {
	return &Cumulus{
		frrApplier:        templates.NewFrrApplier(frrTplFile),
		interfacesApplier: templates.NewInterfacesApplier(interfacesTplFile),
		log:               log,
	}
}

func (c *Cumulus) Apply(cfg *types.Conf) error {
	err := c.interfacesApplier.Apply(cfg)
	if err != nil {
		return err
	}

	return c.frrApplier.Apply(cfg)
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
