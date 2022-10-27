package switcher

import (
	"fmt"
	"github.com/metal-stack/metal-go/api/models"
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

	return c.frrApplier.Apply(cfg)
}

func (c *Cumulus) GetNics(log *zap.SugaredLogger, blacklist []string) (nics []*models.V1SwitchNic, err error) {
	ifs, err := c.GetSwitchPorts()
	if err != nil {
		return nil, fmt.Errorf("unable to get all ifs: %w", err)
	}

	for _, iface := range ifs {
		name := iface.Name
		mac := iface.HardwareAddr.String()
		if contains(blacklist, name) {
			log.Debugw("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}

		if _, err := net.ParseMAC(mac); err != nil {
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
