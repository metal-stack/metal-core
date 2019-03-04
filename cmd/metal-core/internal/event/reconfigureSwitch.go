package event

import (
	"strconv"
	"strings"
	"sync"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/switcher"
	"git.f-i-ts.de/cloud-native/metallib/vlan"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func buildSwitcherConfig(s *models.MetalSwitch, config *domain.Config) (*switcher.Conf, error) {
	c := &switcher.Conf{}
	c.Name = s.Name
	asn64, err := strconv.ParseUint(config.ASN, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}
	c.ASN = asn
	c.Loopback = config.LoopbackIP
	c.Neighbors = strings.Split(config.SpineUplinks, ",")
	c.Tenants = make(map[string]*switcher.Tenant)
	for _, nic := range s.Nics {
		tenant := &switcher.Tenant{}
		if t, has := c.Tenants[nic.Vrf]; has {
			tenant = t
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(nic.Vrf, "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		tenant.VNI = uint32(vni64)
		tenant.Neighbors = append(tenant.Neighbors, *nic.Name)
		c.Tenants[nic.Vrf] = tenant
	}
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

var mux sync.Mutex

func (h *eventHandler) ReconfigureSwitch(switchID string) {
	mux.Lock()
	defer mux.Unlock()
	params := sw.NewFindSwitchParams()
	params.ID = switchID
	fsr, err := h.SwitchClient.FindSwitch(params)
	if err != nil {
		zapup.MustRootLogger().Error("Could not fetch switch from metal-api",
			zap.Error(err))
	}

	s := fsr.Payload
	c, err := buildSwitcherConfig(s, h.Config)
	if err != nil {
		zapup.MustRootLogger().Error("Could build switcher config",
			zap.Error(err))
	}

	zapup.MustRootLogger().Info("Would apply this configuration to the switch",
		zap.Any("config", c))
	err = c.Apply()
	if err != nil {
		zapup.MustRootLogger().Error("Could not apply switch config",
			zap.Error(err))
	}
}
