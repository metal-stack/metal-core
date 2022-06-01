package event

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-core/internal/switcher"
	"github.com/metal-stack/metal-core/internal/vlan"
	"github.com/metal-stack/metal-core/pkg/domain"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

type config struct {
	additionalBridgePorts     []string
	additionalBridgeVIDs      []string
	asn                       string
	cidr                      string
	devMode                   bool
	frrTplFile                string
	interfacesTplFile         string
	logLevel                  string
	loopbackIP                string
	managementGateway         string
	reconfigureSwitch         bool
	reconfigureSwitchInterval time.Duration
	spineUplinks              string
}

func newConfig(cfg *domain.Config, devMode bool) *config {
	return &config{
		additionalBridgePorts:     cfg.AdditionalBridgePorts,
		additionalBridgeVIDs:      cfg.AdditionalBridgeVIDs,
		asn:                       cfg.ASN,
		cidr:                      cfg.CIDR,
		devMode:                   devMode,
		frrTplFile:                cfg.FrrTplFile,
		interfacesTplFile:         cfg.InterfacesTplFile,
		managementGateway:         cfg.ManagementGateway,
		logLevel:                  cfg.LogLevel,
		loopbackIP:                cfg.LoopbackIP,
		reconfigureSwitch:         cfg.ReconfigureSwitch,
		reconfigureSwitchInterval: cfg.ReconfigureSwitchInterval,
		spineUplinks:              cfg.SpineUplinks,
	}
}

// ReconfigureSwitch reconfigures the switch.
func (h *eventHandler) ReconfigureSwitch() {
	t := time.NewTicker(h.config.reconfigureSwitchInterval)
	host, _ := os.Hostname()
	for range t.C {
		h.log.Info("trigger reconfiguration")
		start := time.Now()
		err := h.reconfigureSwitch(host)
		elapsed := time.Since(start)
		h.log.Info("reconfiguration took", zap.Duration("elapsed", elapsed))

		params := sw.NewNotifySwitchParams()
		params.ID = host
		ns := elapsed.Nanoseconds()
		nr := &models.V1SwitchNotifyRequest{
			SyncDuration: &ns,
		}
		if err != nil {
			errStr := err.Error()
			nr.Error = &errStr
			h.log.Error("reconfiguration failed", zap.Error(err))
		} else {
			h.log.Info("reconfiguration succeeded")
		}

		params.Body = nr
		_, err = h.switchClient.NotifySwitch(params, h.auth)
		if err != nil {
			h.log.Error("notification about switch reconfiguration failed", zap.Error(err))
		}
	}
}

func (h *eventHandler) reconfigureSwitch(switchName string) error {
	params := sw.NewFindSwitchParams()
	params.ID = switchName
	fsr, err := h.switchClient.FindSwitch(params, h.auth)
	if err != nil {
		return fmt.Errorf("could not fetch switch from metal-api: %w", err)
	}

	s := fsr.Payload
	c, err := buildSwitcherConfig(h.config, s)
	if err != nil {
		return fmt.Errorf("could not build switcher config: %w", err)
	}

	err = fillEth0Info(c, h.config.managementGateway, h.config.devMode)
	if err != nil {
		return fmt.Errorf("could not gather information about eth0 nic: %w", err)
	}

	h.log.Info("assembled new config for switch",
		zap.Any("config", c))
	if !h.config.reconfigureSwitch {
		h.log.Debug("skip config application because of environment setting")
		return nil
	}

	err = c.Apply()
	if err != nil {
		return fmt.Errorf("could not apply switch config: %w", err)
	}

	return nil
}

func buildSwitcherConfig(conf *config, s *models.V1SwitchResponse) (*switcher.Conf, error) {
	c := &switcher.Conf{}
	c.Name = s.Name
	c.LogLevel = mapLogLevel(conf.logLevel)
	asn64, err := strconv.ParseUint(conf.asn, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}

	c.ASN = asn
	c.Loopback = conf.loopbackIP
	c.MetalCoreCIDR = conf.cidr
	if conf.interfacesTplFile != "" {
		c.InterfacesTplFile = conf.interfacesTplFile
	}
	if conf.frrTplFile != "" {
		c.FrrTplFile = conf.frrTplFile
	}
	c.AdditionalBridgeVIDs = conf.additionalBridgeVIDs
	p := switcher.Ports{
		Underlay:      strings.Split(conf.spineUplinks, ","),
		Unprovisioned: []string{},
		Vrfs:          map[string]*switcher.Vrf{},
		Firewalls:     map[string]*switcher.Firewall{},
	}
	p.BladePorts = conf.additionalBridgePorts
	for _, nic := range s.Nics {
		port := *nic.Name
		if contains(p.Underlay, port) {
			continue
		}
		if contains(conf.additionalBridgePorts, port) {
			continue
		}
		if nic.Vrf == "" {
			if !contains(p.Unprovisioned, port) {
				p.Unprovisioned = append(p.Unprovisioned, port)
			}
			continue
		}
		// Firewall-Port
		if nic.Vrf == "default" {
			fw := &switcher.Firewall{
				Port: port,
			}
			if nic.Filter != nil {
				fw.Vnis = nic.Filter.Vnis
				fw.Cidrs = nic.Filter.Cidrs
			}
			p.Firewalls[port] = fw
			continue
		}
		// Machine-Port
		vrf := &switcher.Vrf{}
		if v, has := p.Vrfs[nic.Vrf]; has {
			vrf = v
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(nic.Vrf, "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		vrf.VNI = uint32(vni64)
		vrf.Neighbors = append(vrf.Neighbors, port)
		if nic.Filter != nil {
			vrf.Cidrs = nic.Filter.Cidrs
		}
		p.Vrfs[nic.Vrf] = vrf
	}
	c.Ports = p
	c.FillRouteMapsAndIPPrefixLists()
	m, err := vlan.ReadMapping()
	if err != nil {
		return nil, err
	}
	err = c.FillVLANIDs(m)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// mapLogLevel maps the metal-core log level to an appropriate FRR log level
// http://docs.frrouting.org/en/latest/basic.html#clicmd-[no]logsyslog[LEVEL]
func mapLogLevel(level string) string {
	switch strings.ToLower(level) {
	case "debug":
		return "debugging"
	case "info":
		return "informational"
	case "warn":
		return "warnings"
	case "error":
		return "errors"
	default:
		return "warnings"
	}
}

func fillEth0Info(c *switcher.Conf, gw string, devMode bool) error {
	c.Ports.Eth0 = switcher.Nic{}
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}
	addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	if len(addrs) < 1 {
		return fmt.Errorf("there is no ip address configured at eth0")
	}

	ip := addrs[0].IP
	s, _ := addrs[0].IPNet.Mask.Size()
	c.Ports.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Ports.Eth0.Gateway = gw
	c.DevMode = devMode
	return nil
}

func contains(l []string, e string) bool {
	for _, i := range l {
		if i == e {
			return true
		}
	}
	return false
}
