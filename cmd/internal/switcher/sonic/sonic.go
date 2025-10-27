package sonic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/metal-stack/metal-core/cmd/internal"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
)

const (
	sonicConfigDBPath = "/etc/sonic/config_db.json"
	SonicVersionFile  = "/etc/sonic/sonic_version.yml"
	redisConfigFile   = "/var/run/redis/sonic-db/database_config.json"
)

type Sonic struct {
	db           *db.DB
	frrApplier   *templates.Applier
	log          *slog.Logger
	redisApplier *redis.Applier
}

type PortInfo struct {
	Alias string `json:"alias"`
}

func New(log *slog.Logger, frrTplFile string) (*Sonic, error) {
	cfg, err := loadRedisConfig(redisConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load database config for SONiC: %w", err)
	}
	sonicDb, err := db.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SONiC databases: %w", err)
	}

	return &Sonic{
		db:           sonicDb,
		frrApplier:   NewFrrApplier(frrTplFile),
		log:          log,
		redisApplier: redis.NewApplier(log, sonicDb),
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

func (s *Sonic) Apply(cfg *types.Conf) error {
	err := s.redisApplier.Apply(cfg)
	if err != nil {
		return err
	}

	return s.frrApplier.Apply(cfg)
}

func (s *Sonic) IsInitialized() (initialized bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.db.Appl.ExistPortInitDone(ctx)
}

func (s *Sonic) GetNics(log *slog.Logger, blacklist []string) (nics []*models.V1SwitchNic, err error) {
	ports, err := getPortsConfig(sonicConfigDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get ports config")
	}

	for name, portConfig := range ports {
		if slices.Contains(blacklist, name) {
			log.Debug("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}

		nic := &models.V1SwitchNic{
			Identifier: &portConfig.Alias,
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
	ports, err := getPortsConfig(sonicConfigDBPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	return portsToInterfaces(ports), nil
}

func portsToInterfaces(ports map[string]PortInfo) []*net.Interface {
	interfaces := make([]*net.Interface, 0)

	for portName := range ports {
		interfaces = append(interfaces, &net.Interface{
			Name: portName,
		})
	}
	slices.SortStableFunc(interfaces, func(a, b *net.Interface) int {
		return strings.Compare(a.Name, b.Name)
	})

	return interfaces
}

func getPortsConfig(filepath string) (map[string]PortInfo, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = jsonFile.Close()
	}()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	config := struct {
		Ports map[string]PortInfo `json:"PORT"`
	}{}
	//nolint:musttag
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
	// problem
	ip, err = internal.GetManagementIP("Ethernet0")
	if err != nil {
		return "", "", err
	}
	return ip, "admin", nil
}
