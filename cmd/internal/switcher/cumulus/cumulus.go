package cumulus

import (
	"fmt"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/metal-stack/metal-core/cmd/internal"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
)

type Cumulus struct {
	frrApplier        *templates.Applier
	interfacesApplier *templates.Applier
	log               *zap.SugaredLogger
}

func New(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) *Cumulus {
	return &Cumulus{
		frrApplier:        NewFrrApplier(frrTplFile),
		interfacesApplier: NewInterfacesApplier(interfacesTplFile),
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

func (c *Cumulus) GetNics(log *zap.SugaredLogger, blacklist []string) (nics []*models.V1SwitchNic, err error) {
	ifs, err := c.GetSwitchPorts()
	if err != nil {
		return nil, fmt.Errorf("unable to get all ifs: %w", err)
	}

	for _, iface := range ifs {
		name := iface.Name
		mac := iface.HardwareAddr.String()
		if slices.Contains(blacklist, name) {
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
			c.log.Debugw("skip interface, because only swp* interface are front panels", "interface", iface.Name)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	return switchPorts, nil
}

func (c *Cumulus) SanitizeConfig(cfg *types.Conf) {
	// nothing required here
}

func (c *Cumulus) GetOS() (*models.V1SwitchOS, error) {
	version := "unknown"
	lsbReleaseBytes, err := os.ReadFile("/etc/lsb-release")
	if err != nil {
		c.log.Errorw("unable to read /etc/lsb-release", "error", err)
	} else {
		for _, line := range strings.Fields(string(lsbReleaseBytes)) {
			if strings.HasPrefix(line, "DISTRIB_RELEASE") {
				_, v, found := strings.Cut(line, "=")
				if found {
					version = v
				}
			}
		}
	}
	return &models.V1SwitchOS{
		Vendor:  "Cumulus",
		Version: version,
	}, nil
}
func (c *Cumulus) GetManagement() (ip, user string, err error) {
	ip, err = internal.GetManagementIP("eth0")
	if err != nil {
		return "", "", err
	}
	return ip, "cumulus", nil
}
