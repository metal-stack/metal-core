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

// ReconfigureSwitch reconfigures the switch.
func (h *eventHandler) ReconfigureSwitch() {
	t := time.NewTicker(h.AppContext.Config.ReconfigureSwitchInterval)
	host, _ := os.Hostname()
	for range t.C {
		h.Log.Info("trigger reconfiguration")
		start := time.Now()
		err := h.reconfigureSwitch(host)
		elapsed := time.Since(start)
		h.Log.Info("reconfiguration took", zap.Duration("elapsed", elapsed))

		params := sw.NewNotifySwitchParams()
		params.ID = host
		ns := elapsed.Nanoseconds()
		nr := &models.V1SwitchNotifyRequest{
			SyncDuration: &ns,
		}
		if err != nil {
			errStr := err.Error()
			nr.Error = &errStr
			h.Log.Error("reconfiguration failed", zap.Error(err))
		} else {
			h.Log.Info("reconfiguration succeeded")
		}

		params.Body = nr
		_, err = h.SwitchClient.NotifySwitch(params, h.Auth)
		if err != nil {
			h.Log.Error("notification about switch reconfiguration failed", zap.Error(err))
		}
	}
}

func (h *eventHandler) reconfigureSwitch(switchName string) error {
	params := sw.NewFindSwitchParams()
	params.ID = switchName
	fsr, err := h.SwitchClient.FindSwitch(params, h.Auth)
	if err != nil {
		return fmt.Errorf("could not fetch switch from metal-api: %w", err)
	}

	s := fsr.Payload
	c, err := buildSwitcherConfig(h.Config, s)
	if err != nil {
		return fmt.Errorf("could not build switcher config: %w", err)
	}

	err = fillEth0Info(c, h.Config.ManagementGateway, h.DevMode)
	if err != nil {
		return fmt.Errorf("could not gather information about eth0 nic: %w", err)
	}

	h.Log.Info("assembled new config for switch",
		zap.Any("config", c))
	if !h.Config.ReconfigureSwitch {
		h.Log.Debug("skip config application because of environment setting")
		return nil
	}

	err = c.Apply()
	if err != nil {
		return fmt.Errorf("could not apply switch config: %w", err)
	}

	return nil
}

func buildSwitcherConfig(conf *domain.Config, s *models.V1SwitchResponse) (*switcher.Conf, error) {
	c := &switcher.Conf{}
	c.Name = s.Name
	c.LogLevel = mapLogLevel(conf.LogLevel)
	asn64, err := strconv.ParseUint(conf.ASN, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}

	c.ASN = asn
	c.Loopback = conf.LoopbackIP
	c.MetalCoreCIDR = conf.CIDR
	if conf.InterfacesTplFile != "" {
		c.InterfacesTplFile = conf.InterfacesTplFile
	}
	if conf.FrrTplFile != "" {
		c.FrrTplFile = conf.FrrTplFile
	}
	c.AdditionalBridgeVIDs = conf.AdditionalBridgeVIDs
	p := switcher.Ports{
		Underlay:      strings.Split(conf.SpineUplinks, ","),
		Unprovisioned: []string{},
		Vrfs:          map[string]*switcher.Vrf{},
		Firewalls:     map[string]*switcher.Firewall{},
	}
	p.BladePorts = conf.AdditionalBridgePorts
	for _, nic := range s.Nics {
		port := *nic.Name
		if contains(p.Underlay, port) {
			continue
		}
		if contains(conf.AdditionalBridgePorts, port) {
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
