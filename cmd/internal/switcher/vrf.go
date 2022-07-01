package switcher

import "fmt"

const (
	vlanTable           = "VLAN"
	vlanInterfaceTable  = "VLAN_INTERFACE"
	vrfTable            = "VRF"
	vxlanTunnelMapTable = "VXLAN_TUNNEL_MAP"

	untagged = "untagged"
)

type vrf struct {
	name string
	vni  string
}

func applyVrfs(db *ConfigDB, vrfs []*vrf) error {
	view, err := db.GetView(vrfTable)
	if err != nil {
		return err
	}

	for _, vrf := range vrfs {
		key := []string{vrfTable, vrf.name}
		if view.Contains(key) {
			view.Mask(key)
			entry, err := db.GetEntry(key)
			if err != nil {
				return err
			}
			if entry["vni"] == vrf.vni {
				continue
			}
		}
		err = db.SetEntry(key, "vni", vrf.vni)
		if err != nil {
			return err
		}
	}
	return view.DeleteUnmasked()
}

func applyVlan(db *ConfigDB, vlanIds []string) error {
	view, err := db.GetView(vlanTable)
	if err != nil {
		return err
	}

	for _, vlanId := range vlanIds {
		vlanName := "Vlan" + vlanId
		key := []string{vlanTable, vlanName}
		if view.Contains(key) {
			view.Mask(key)
			continue
		}
		err = db.SetEntry(key, "vlanid", vlanId)
		if err != nil {
			return err
		}
	}
	return view.DeleteUnmasked()
}

type vlanIface struct {
	cidr    string
	name    string
	vrfName string
}

func applyVlanInterfaces(db *ConfigDB, vlanIfaces []*vlanIface) error {
	view, err := db.GetView(vlanInterfaceTable)
	if err != nil {
		return err
	}

	for _, iface := range vlanIfaces {
		if iface.cidr != "" {
			key := []string{vlanInterfaceTable, iface.name, iface.cidr}
			if view.Contains(key) {
				view.Mask(key)
			} else {
				err = db.SetEntry(key)
				if err != nil {
					return err
				}
			}
		}

		key := []string{vlanInterfaceTable, iface.name}
		if view.Contains(key) {
			view.Mask(key)
			entry, err := db.GetEntry(key)
			if err != nil {
				return err
			}
			vrfName, ok := entry["vrf_name"]
			if (!ok && iface.vrfName == "") || (ok && vrfName == iface.vrfName) {
				continue
			}
		}
		if iface.vrfName == "" {
			err = db.SetEntry(key)
		} else {
			err = db.SetEntry(key, "vrf_name", iface.vrfName)
		}
		if err != nil {
			return err
		}
	}
	return view.DeleteUnmasked()
}

var vlanMemberTable = "VLAN_MEMBER"

func applyVlanMember(db *ConfigDB, vlan string, members []string) error {
	view, err := db.GetView(vlanMemberTable)
	if err != nil {
		return err
	}

	for _, member := range members {
		key := []string{vlanMemberTable, vlan, member}
		if view.Contains(key) {
			view.Mask(key)
			continue
		}
		err = db.SetEntry(key, "tagging_mode", untagged)
		if err != nil {
			return err
		}
	}
	return view.DeleteUnmasked()
}

type vxlanTunnelMap struct {
	vlan string
	vni  string
}

func applyVxlanTunnelMap(db *ConfigDB, tunnels []*vxlanTunnelMap) error {
	view, err := db.GetView(vxlanTunnelMapTable)
	if err != nil {
		return err
	}

	for _, tunnel := range tunnels {
		name := fmt.Sprintf("vtep|map_%s_%s", tunnel.vni, tunnel.vlan)
		key := []string{vxlanTunnelMapTable, name}
		if view.Contains(key) {
			view.Mask(key)
			continue
		}
		err = db.SetEntry(key, "vlan", tunnel.vlan, "vni", tunnel.vni)
		if err != nil {
			return err
		}
	}
	return view.DeleteUnmasked()
}
