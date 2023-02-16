package configdb

type redis struct {
}

func (r *redis) setVLANMember(interfaceName, vlan string) error {
	return nil
}

func (r *redis) deleteVLANMember(interfaceName string, vlan uint16) error {
	return nil
}

func (r *redis) setVRFMember(interfaceName string, vrf string) error {
	return nil
}

func (r *redis) deleteVRFMember(interfaceName string) error {
	return nil
}
