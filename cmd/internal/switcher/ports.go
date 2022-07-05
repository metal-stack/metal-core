package switcher

const (
	interfaceTable = "INTERFACE"
	portTable      = "PORT"
)

type port struct {
	mtu     string
	vrfName string
}

func getPorts(cfg *Conf) map[string]*port {
	ports := make(map[string]*port)

	for _, p := range cfg.Ports.Underlay {
		ports[p] = &port{mtu: "9216"}
	}
	for _, fw := range cfg.Ports.Firewalls {
		ports[fw.Port] = &port{mtu: "9216"}
	}
	for vrfName, v := range cfg.Ports.Vrfs {
		for _, p := range v.Neighbors {
			ports[p] = &port{vrfName: vrfName, mtu: "9000"}
		}
	}
	for _, p := range cfg.Ports.Unprovisioned {
		ports[p] = &port{mtu: "9000"}
	}

	return ports
}

func applyMtus(db *ConfigDB, ports map[string]*port) error {
	for iface, p := range ports {
		err := db.ModEntry([]string{portTable, iface}, "mtu", p.mtu)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyPorts(db *ConfigDB, ports map[string]*port) error {
	view, err := db.GetView(interfaceTable)
	if err != nil {
		return err
	}

	for iface, p := range ports {
		if p.vrfName == "" {
			continue
		}

		key := []string{interfaceTable, iface}
		if view.Contains(key) {
			view.Mask(key)
			entry, err := db.GetEntry(key)
			if err != nil {
				return err
			}
			if entry["vrf_name"] == p.vrfName {
				continue
			}
		}
		err = db.SetEntry(key, "vrf_name", p.vrfName)
		if err != nil {
			return err
		}
	}
	return nil //view.DeleteUnmasked()
}
