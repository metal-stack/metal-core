package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

// ReconfigureSwitch reconfigures the switch.
func (c *Core) ReconfigureSwitch() {
	t := time.NewTicker(c.reconfigureSwitchInterval)
	host, _ := os.Hostname()
	for range t.C {
		c.log.Info("trigger reconfiguration")
		start := time.Now()
		err := c.reconfigureSwitch(host)
		elapsed := time.Since(start)
		c.log.Info("reconfiguration took", zap.Duration("elapsed", elapsed))

		params := sw.NewNotifySwitchParams()
		params.ID = host
		ns := elapsed.Nanoseconds()
		nr := &models.V1SwitchNotifyRequest{
			SyncDuration: &ns,
		}
		if err != nil {
			errStr := err.Error()
			nr.Error = &errStr
			c.log.Error("reconfiguration failed", zap.Error(err))
		} else {
			c.log.Info("reconfiguration succeeded")
		}

		params.Body = nr
		_, err = c.driver.SwitchOperations().NotifySwitch(params, nil)
		if err != nil {
			c.log.Error("notification about switch reconfiguration failed", zap.Error(err))
		}
	}
}

func (c *Core) reconfigureSwitch(switchName string) error {
	params := sw.NewFindSwitchParams()
	params.ID = switchName
	fsr, err := c.driver.SwitchOperations().FindSwitch(params, nil)
	if err != nil {
		return fmt.Errorf("could not fetch switch from metal-api: %w", err)
	}

	s := fsr.Payload
	switchConfig, err := c.buildSwitcherConfig(s)
	if err != nil {
		return fmt.Errorf("could not build switcher config: %w", err)
	}

	err = fillEth0Info(switchConfig, c.managementGateway)
	if err != nil {
		return fmt.Errorf("could not gather information about eth0 nic: %w", err)
	}

	c.log.Info("assembled new config for switch",
		zap.Any("config", switchConfig))
	if !c.enableReconfigureSwitch {
		c.log.Debug("skip config application because of environment setting")
		return nil
	}

	err = switchConfig.Apply()
	if err != nil {
		return fmt.Errorf("could not apply switch config: %w", err)
	}

	return nil
}

func (c *Core) buildSwitcherConfig(s *models.V1SwitchResponse) (*switcher.Conf, error) {
	asn64, err := strconv.ParseUint(c.asn, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}
	switcherConfig := &switcher.Conf{
		Name:                 s.Name,
		LogLevel:             mapLogLevel(c.logLevel),
		ASN:                  asn,
		Loopback:             c.loopbackIP,
		MetalCoreCIDR:        c.cidr,
		AdditionalBridgeVIDs: c.additionalBridgeVIDs,
	}

	if c.interfacesTplFile != "" {
		switcherConfig.InterfacesTplFile = c.interfacesTplFile
	}
	if c.frrTplFile != "" {
		switcherConfig.FrrTplFile = c.frrTplFile
	}
	p := switcher.Ports{
		Underlay:      strings.Split(c.spineUplinks, ","),
		Unprovisioned: []string{},
		Vrfs:          map[string]*switcher.Vrf{},
		Firewalls:     map[string]*switcher.Firewall{},
	}
	p.BladePorts = c.additionalBridgePorts
	for _, nic := range s.Nics {
		port := *nic.Name
		if contains(p.Underlay, port) {
			continue
		}
		if contains(c.additionalBridgePorts, port) {
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
	switcherConfig.Ports = p
	switcherConfig.FillRouteMapsAndIPPrefixLists()
	m, err := vlan.ReadMapping()
	if err != nil {
		return nil, err
	}
	err = switcherConfig.FillVLANIDs(m)
	if err != nil {
		return nil, err
	}
	return switcherConfig, nil
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

func fillEth0Info(c *switcher.Conf, gw string) error {
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
