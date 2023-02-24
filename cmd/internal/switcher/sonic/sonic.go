package sonic

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/metal-stack/metal-core/cmd/internal"
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"

	"go.uber.org/zap"
)

const (
	sonicConfigDBPath = "/etc/sonic/config_db.json"
	SonicVersionFile  = "/etc/sonic/sonic_version.yml"

	frr                  = "/etc/sonic/frr/frr.conf"
	frrTmp               = "/etc/sonic/frr/frr.tmp"
	frrReloadService     = "frr-reload.service"
	frrValidationService = "bgp-validation"

	redisConfigFile = "/var/run/redis/sonic-db/database_config.json"
)

var frrTpl = "sonic_frr.tpl"

type Sonic struct {
	frrApplier   *templates.FrrApplier
	redisApplier *redis.Applier
	log          *zap.SugaredLogger
}

type PortInfo struct {
	Alias string
}

func New(log *zap.SugaredLogger, frrTplFile string) (*Sonic, error) {
	log.Infow("create sonic NOS")

	embedFS := true
	if frrTplFile != "" {
		frrTpl = frrTplFile
		embedFS = false
	}

	cfg, err := loadRedisConfig(redisConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load database config for SONiC: %w", err)
	}

	return &Sonic{
		frrApplier:   templates.NewFrrApplier(frr, frrTmp, frrValidationService, "", frrTpl, embedFS),
		redisApplier: redis.NewApplier(log, cfg),
		log:          log,
	}, nil
}

func loadRedisConfig(path string) (*db.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &db.Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *Sonic) Apply(cfg *types.Conf) (updated bool, err error) {
	redisApplied, err := s.redisApplier.Apply(cfg)
	if err != nil {
		return false, err
	}

	frrApplied, err := s.frrApplier.Apply(cfg)
	if err != nil {
		return false, err
	}

	// TODO should be moved to frrApplier
	if frrApplied {
		if err := dbus.Start(frrReloadService); err != nil {
			return false, fmt.Errorf("failed reload FRR %w", err)
		}
	}

	return frrApplied || redisApplied, nil
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
