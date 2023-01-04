package sonic

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal"
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"go.uber.org/zap"
)

const (
	sonicConfigDBPath            = "/etc/sonic/config_db.json"
	sonicConfigSaveReloadService = "config-save-reload.service"
	SonicVersionFile             = "/etc/sonic/sonic_version.yml"

	frr                  = "/etc/sonic/frr/frr.conf"
	frrTmp               = "/etc/sonic/frr/frr.tmp"
	frrValidationService = "bgp-validation"
)

var frrTpl = "sonic_frr.tpl"

type Sonic struct {
	frrApplier     *templates.FrrApplier
	confidbApplier *templates.ConfigdbApplier
	log            *zap.SugaredLogger
}

type PortInfo struct {
	Alias string
}

func New(log *zap.SugaredLogger, frrTplFile string) (*Sonic, error) {
	ifs, err := getInterfacesConfig(sonicConfigDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load interfaces config from configDB: %w", err)
	}

	embedFS := true
	if frrTplFile != "" {
		frrTpl = frrTplFile
		embedFS = false
	}

	return &Sonic{
		frrApplier:     templates.NewFrrApplier(frr, frrTmp, frrValidationService, "", frrTpl, embedFS),
		confidbApplier: templates.NewConfigdbApplier(ifs),
		log:            log,
	}, nil
}

func (s *Sonic) Apply(cfg *types.Conf) error {
	bgpApplied, err := s.frrApplier.Apply(cfg)
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
		if slices.Contains(blacklist, name) {
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

func (s *Sonic) SanitizeConfig(cfg *types.Conf) {
	cfg.CapitalizeVrfName()
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
			s.log.Debugw("skip interface, because only Ethernet* interface are front panels", "interface", iface.Name)
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
	err = json.Unmarshal(byteValue, &config)

	return config.Ports, err
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
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return nil, err
	}

	for k, _ := range config.Interfaces {
		infs = append(infs, k)
	}

	return infs, nil
}

type sonic_version struct {
	BuildVersion string `yaml:"build_version"`
}

func (s *Sonic) GetOS() (*models.V1SwitchOS, error) {
	versionBytes, err := os.ReadFile(SonicVersionFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read sonic_version: %w", err)
	}

	var sonicVersion sonic_version
	err = yaml.Unmarshal(versionBytes, &sonicVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse sonic_version: %w", err)
	}
	return &models.V1SwitchOS{
		Vendor:  "SONiC",
		Version: sonicVersion.BuildVersion,
	}, nil
}
func (s *Sonic) GetManagement() (ip, user string, err error) {
	ip, err = internal.GetManagementIP("eth0")
	if err != nil {
		return "", "", err
	}
	return ip, "admin", nil
}
