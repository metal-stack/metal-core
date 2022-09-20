package switcher

import (
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const configDB = "CONFIG_DB"

type SonicDatabaseConfig struct {
	Instances map[string]Instance `json:"INSTANCES"`
	Databases map[string]Database `json:"DATABASES"`
	Version   string              `json:"VERSION"`
}

type Instance struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

type Database struct {
	Id        int    `json:"id"`
	Separator string `json:"separator"`
	Instance  string `json:"instance"`
}

type Sonic struct {
	bgpApplier     *BgpApplier
	confidbApplier *ConfigDBApplier
	log            *zap.SugaredLogger
}

func NewSonic(cfg *SonicDatabaseConfig, log *zap.SugaredLogger) *Sonic {
	return &Sonic{
		bgpApplier:     newBgpApplier(),
		confidbApplier: NewConfigDBApplier(cfg),
		log:            log,
	}
}

func (s *Sonic) Apply(cfg *Conf) error {
	c := capitalizeVrfName(cfg)
	err := s.bgpApplier.Apply(c)
	if err != nil {
		return err
	}
	return s.confidbApplier.Apply(c)
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
				zap.String("MAC", iface.HardwareAddr.String()),
			)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	return switchPorts, nil
}

func capitalizeVrfName(cfg *Conf) *Conf {
	caser := cases.Title(language.English)
	vrfs := make(map[string]*Vrf)
	for name, vrf := range cfg.Ports.Vrfs {
		s := caser.String(name)
		vrfs[s] = vrf
	}
	p := Ports{
		Eth0:          cfg.Ports.Eth0,
		Underlay:      cfg.Ports.Underlay,
		Unprovisioned: cfg.Ports.Unprovisioned,
		BladePorts:    cfg.Ports.BladePorts,
		Vrfs:          vrfs,
		Firewalls:     cfg.Ports.Firewalls,
	}
	return &Conf{
		Name:                 cfg.Name,
		LogLevel:             cfg.LogLevel,
		Loopback:             cfg.Loopback,
		ASN:                  cfg.ASN,
		Ports:                p,
		MetalCoreCIDR:        cfg.MetalCoreCIDR,
		AdditionalBridgeVIDs: cfg.AdditionalBridgeVIDs,
	}
}
