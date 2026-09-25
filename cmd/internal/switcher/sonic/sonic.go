package sonic

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal"
	corenet "github.com/metal-stack/metal-core/cmd/internal/net"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type (
	Sonic struct {
		db                    *db.DB
		frrApplier            *templates.Applier
		log                   *slog.Logger
		redisApplier          *redis.Applier
		interfaceNamingSchema InterfaceNamingSchema
	}

	PortInfo struct {
		Alias string `json:"alias"`
	}

	InterfaceNamingSchema string
)

const (
	InterfaceNamingSchemaDefault = InterfaceNamingSchema("default")
	InterfaceNamingSchemaSwap    = InterfaceNamingSchema("swap")
	InterfaceNamingSchemaName    = InterfaceNamingSchema("name")
	InterfaceNamingSchemaAlias   = InterfaceNamingSchema("alias")
)

const (
	SonicVersionFile = "/etc/sonic/sonic_version.yml"
	redisConfigFile  = "/var/run/redis/sonic-db/database_config.json"
)

func New(log *slog.Logger, frrTplFile string, interfaceNamingSchema InterfaceNamingSchema) (*Sonic, error) {
	cfg, err := loadRedisConfig(redisConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load database config for SONiC: %w", err)
	}
	sonicDb, err := db.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SONiC databases: %w", err)
	}

	return &Sonic{
		db:                    sonicDb,
		frrApplier:            NewFrrApplier(log, frrTplFile),
		log:                   log,
		redisApplier:          redis.NewApplier(log, sonicDb),
		interfaceNamingSchema: interfaceNamingSchema,
	}, nil
}

func (s *Sonic) Apply(ctx context.Context, cfg *types.Conf) error {
	err := s.redisApplier.Apply(ctx, cfg)
	if err != nil {
		return err
	}

	return s.frrApplier.Apply(ctx, cfg)
}

func (s *Sonic) IsInitialized(ctx context.Context) (initialized bool, err error) {
	return s.db.Appl.ExistPortInitDone(ctx)
}

func (s *Sonic) GetNics(ctx context.Context, blacklist []string) (nics []*apiv2.SwitchNic, err error) {
	ports, err := s.getPortsConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ports config")
	}

	for name, portConfig := range ports {
		if slices.Contains(blacklist, name) {
			s.log.Debug("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}

		linkStatus, err := corenet.GetLinkStatus(name)
		if err != nil {
			s.log.Error("failed to get link status", "port", name, "status", linkStatus, "error", err)
		}

		nic := getSwitchNicByNamingSchema(name, portConfig.Alias, s.interfaceNamingSchema, linkStatus)
		nics = append(nics, nic)
	}

	return nics, nil
}

func (s *Sonic) SanitizeConfig(cfg *types.Conf) {
	cfg.CapitalizeVrfName()
}

func (s *Sonic) GetPortOrdinals(ctx context.Context) (map[string]int, error) {
	ports, err := s.redisApplier.GetPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get ports: %w", err)
	}
	return portOrdinals(ports)
}

// portOrdinals orders the ports by their physical layout (port index, then first asic lane) and returns the position of every port.
// The Ethernet<n> naming must not be used because it is platform dependent (broadcom counts lanes: Ethernet0, Ethernet4, ...)
// and the index alone is not unique when ports are broken out.
func portOrdinals(ports []*db.Port) (map[string]int, error) {
	type sortablePort struct {
		name  string
		index int
		lane  int
	}

	sortable := make([]sortablePort, 0, len(ports))
	for _, port := range ports {
		index, err := strconv.Atoi(port.Index)
		if err != nil {
			return nil, fmt.Errorf("port %s has no numeric index %q in CONFIG_DB: %w", port.Name, port.Index, err)
		}
		firstLane, _, _ := strings.Cut(port.Lanes, ",")
		lane, err := strconv.Atoi(strings.TrimSpace(firstLane))
		if err != nil {
			return nil, fmt.Errorf("port %s has no numeric lanes %q in CONFIG_DB: %w", port.Name, port.Lanes, err)
		}
		sortable = append(sortable, sortablePort{name: port.Name, index: index, lane: lane})
	}

	slices.SortFunc(sortable, func(a, b sortablePort) int {
		if a.index != b.index {
			return a.index - b.index
		}
		if a.lane != b.lane {
			return a.lane - b.lane
		}
		return strings.Compare(a.name, b.name)
	})

	ordinals := make(map[string]int, len(sortable))
	for i, port := range sortable {
		ordinals[port.name] = i
	}
	return ordinals, nil
}

func (s *Sonic) GetSwitchPorts(ctx context.Context) ([]*net.Interface, error) {
	ports, err := s.getPortsConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	return portsToInterfaces(ports), nil
}

func (s *Sonic) GetOS() (*apiv2.SwitchOS, error) {
	versionBytes, err := os.ReadFile(SonicVersionFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read sonic_version: %w", err)
	}

	var sonicVersion struct {
		BuildVersion string `yaml:"build_version"`
	}
	err = yaml.Unmarshal(versionBytes, &sonicVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse sonic_version: %w", err)
	}
	return &apiv2.SwitchOS{
		Vendor:  apiv2.SwitchOSVendor_SWITCH_OS_VENDOR_SONIC,
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

func (s *Sonic) getPortsConfig(ctx context.Context) (map[string]PortInfo, error) {
	ports, err := s.redisApplier.GetPorts(ctx)
	if err != nil {
		return nil, err
	}

	// keep the real interface names as keys; the naming schema is only applied
	// when reporting nics to the metal-api. The keys are used to match the
	// interface blacklist and to open the LLDP pcap handles, both of which
	// need the actual netdev names.
	portConfig := map[string]PortInfo{}
	for _, port := range ports {
		portConfig[port.Name] = PortInfo{
			Alias: port.Alias,
		}
	}

	return portConfig, err
}

func getSwitchNicByNamingSchema(name, alias string, naming InterfaceNamingSchema, linkStatus apiv2.SwitchPortStatus) *apiv2.SwitchNic {
	var nic = &apiv2.SwitchNic{
		State: &apiv2.NicState{
			Actual: linkStatus,
		},
	}

	switch naming {
	case InterfaceNamingSchemaDefault:
		nic.Name = name
		nic.Identifier = alias
	case InterfaceNamingSchemaSwap:
		nic.Name = alias
		nic.Identifier = name
	case InterfaceNamingSchemaAlias:
		nic.Name = alias
		nic.Identifier = alias
	case InterfaceNamingSchemaName:
		nic.Name = name
		nic.Identifier = name
	default:
		nic.Name = name
		nic.Identifier = alias
	}
	return nic
}
