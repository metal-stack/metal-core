package switcher

import (
	"encoding/json"
	"fmt"
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-go/api/models"
	"io"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"
)

const (
	sonicConfigDBPath            = "/etc/sonic/config_db.json"
	sonicConfigSaveReloadService = "config-save-reload.service"
)

type Sonic struct {
	bgpApplier     *BgpApplier
	confidbApplier *ConfigdbApplier
	log            *zap.SugaredLogger
}

type PortInfo struct {
	Alias string
}

func NewSonic(log *zap.SugaredLogger) (*Sonic, error) {
	ifs, err := getInterfacesConfig(sonicConfigDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load interfaces config from ConfgiDB: %w", err)
	}

	return &Sonic{
		bgpApplier:     newBgpApplier(),
		confidbApplier: newConfigdbApplier(ifs),
		log:            log,
	}, nil
}

func (s *Sonic) Apply(cfg *Conf) error {
	cfg.CapitalizeVrfName()
	bgpApplied, err := s.bgpApplier.Apply(cfg)
	if err != nil {
		return err
	}

	configDBApplied, err := s.confidbApplier.Apply(cfg)
	if err != nil {
		return err
	}

	// Save ConfiDB and reload configuration
	if bgpApplied || configDBApplied {
		if err := dbus.Start(sonicConfigSaveReloadService); err != nil {
			return fmt.Errorf("failed to save and reload SONiC config")
		}
	}

	return nil
}

func (s *Sonic) GetNics(log *zap.SugaredLogger, blacklist []string) (nics []*models.V1SwitchNic, err error) {
	ifs, err := s.GetSwitchPorts()
	if err != nil {
		return nil, fmt.Errorf("unable to get all ifs: %w", err)
	}

	portsConfig, err := getPortsConfig(sonicConfigDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get ports config")
	}

	for _, iface := range ifs {
		name := iface.Name
		if contains(blacklist, name) {
			log.Debugw("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}

		id, found := portsConfig[name]
		if !found {
			log.Debugw("skip interface as no info on it was found in config DB", "interface", name)
			continue
		}

		nic := &models.V1SwitchNic{
			Identifier: &id.Alias,
			Name:       &name,
		}
		nics = append(nics, nic)
	}

	return nics, nil
}

func (s *Sonic) GetSwitchPorts() ([]*net.Interface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	switchPorts := make([]*net.Interface, 0, len(ifs))
	for i := range ifs {
		iface := &ifs[i]
		if !strings.HasPrefix(iface.Name, "Ethernet") {
			s.log.Debug("skip interface, because only Ethernet* interface are front panels",
				zap.String("interface", iface.Name),
			)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	return switchPorts, nil
}

func getPortsConfig(filepath string) (map[string]PortInfo, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	config := struct {
		Ports map[string]PortInfo `json:"PORT"`
	}{}
	json.Unmarshal(byteValue, &config)

	return config.Ports, nil
}

func getInterfacesConfig(filepath string) (infs []string, err error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	config := struct {
		Interfaces map[string]struct{} `json:"INTERFACE"`
	}{}
	json.Unmarshal(byteValue, &config)

	for k, _ := range config.Interfaces {
		infs = append(infs, k)
	}

	return infs, nil
}
