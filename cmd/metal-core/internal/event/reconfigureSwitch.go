package event

import (
	"strconv"
	"strings"
	"sync"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/switcher"
	"git.f-i-ts.de/cloud-native/metallib/vlan"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) buildSwitcherConfig(s *models.MetalSwitch) (*switcher.Conf, error) {
	c := &switcher.Conf{}
	c.Name = s.Name
	asn64, err := strconv.ParseUint(h.Config.ASN, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}
	c.ASN = asn
	c.Loopback = h.Config.LoopbackIP
	c.Neighbors = strings.Split(h.Config.SpineUplinks, ",")
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
	c, err := h.buildSwitcherConfig(s)
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
