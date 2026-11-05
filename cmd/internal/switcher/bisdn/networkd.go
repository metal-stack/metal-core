package bisdn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/vishvananda/netlink"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const networkdDir = "/etc/systemd/network"

// managedFile matches the networkd files that belong to metal-core,
// NN-metal-core-<interface>.<kind>. Other files in the directory, such as the
// mgmt VRF and the management port, are left alone.
var managedFile = regexp.MustCompile(`^[0-9]{2}-metal-core-([A-Za-z0-9._-]+)\.(netdev|network)$`)

type (
	// links changes kernel netdevs where networkd can't: it never deletes a
	// netdev or releases a port from its master when a file goes away.
	links interface {
		Delete(name string) error
		Release(name string) error
	}

	networkdApplier struct {
		dir     string
		tpl     *template.Template
		links   links
		reload  func(ctx context.Context) error
		log     *slog.Logger
		applied bool
	}

	// iface is the input of a networkd template: one interface, rendered
	// into one file.
	iface struct {
		Name                string
		Down                bool
		VRF                 string
		VNI                 uint32
		VLANID              uint16
		Loopback            string
		Address             string
		NeighborSuppression bool
		// bridge only: the SVIs on top of it and its VLANs
		SVIs  []string
		VLANs []string
	}

	netlinkLinks struct{}
)

func newNetworkdApplier(log *slog.Logger, tpl *template.Template) *networkdApplier {
	return &networkdApplier{
		dir:    networkdDir,
		tpl:    tpl,
		links:  netlinkLinks{},
		reload: dbus.ReloadNetworkd,
		log:    log,
	}
}

// Apply renders the networkd files, removes kernel state that the new files no
// longer describe, installs the files and reloads systemd-networkd.
func (a *networkdApplier) Apply(ctx context.Context, cfg *types.Conf) error {
	desired, err := render(a.tpl, cfg)
	if err != nil {
		return err
	}

	current, err := readManagedFiles(a.dir)
	if err != nil {
		return err
	}

	changed := !maps.EqualFunc(current, desired, bytes.Equal)
	if !changed && a.applied {
		a.log.Debug("networkd files are unchanged, nothing to do")
		return nil
	}
	a.applied = false

	if changed {
		// Clean up before the files are replaced: if a step fails, the old
		// files are still there and the next run computes the same changes.
		if err := a.cleanup(current, desired); err != nil {
			return err
		}
		if err := writeFiles(a.dir, current, desired); err != nil {
			return err
		}
	}

	a.log.Debug("reload systemd-networkd")
	if err := a.reload(ctx); err != nil {
		return err
	}
	a.applied = true
	return nil
}

// render returns the networkd files for cfg by file name.
func render(tpl *template.Template, cfg *types.Conf) (map[string][]byte, error) {
	files := map[string][]byte{}
	seen := map[string]bool{}
	var errs []error
	add := func(prio int, kind string, i iface) {
		name := fmt.Sprintf("%02d-metal-core-%s%s", prio, i.Name, path.Ext(kind))
		if !managedFile.MatchString(name) {
			errs = append(errs, fmt.Errorf("invalid interface name %q", i.Name))
			return
		}
		if seen[i.Name+path.Ext(kind)] {
			errs = append(errs, fmt.Errorf("interface %s is configured twice", i.Name))
			return
		}
		seen[i.Name+path.Ext(kind)] = true
		var b bytes.Buffer
		if err := tpl.ExecuteTemplate(&b, kind+".tpl", i); err != nil {
			errs = append(errs, fmt.Errorf("failed to render %s: %w", name, err))
			return
		}
		files[name] = []byte(strings.TrimSpace(b.String()) + "\n")
	}
	down := func(port string) bool {
		return cfg.Ports.AdminStatus[port] == types.PortStatusDown
	}
	// netdev renders a netdev and its .network file
	netdev := func(prio int, kind string, i iface) {
		add(prio, kind+".netdev", i)
		add(prio, kind+".network", i)
	}
	vxlan := func(vni uint32, vlan uint16, suppress bool) {
		netdev(42, "vxlan", iface{Name: fmt.Sprintf("vni%d", vni), VNI: vni, VLANID: vlan, Loopback: cfg.Loopback, NeighborSuppression: suppress})
	}

	add(10, "lo.network", iface{Name: "lo", Address: cfg.Loopback})

	underlay := slices.Clone(cfg.Ports.Underlay)
	for _, fw := range cfg.Ports.Firewalls {
		underlay = append(underlay, fw.Port)
	}
	for _, port := range underlay {
		add(20, "underlay.network", iface{Name: port, Down: down(port)})
	}

	// PXE: SVI with metal-core's address and L2VNI 10<vlan>
	pxe := cfg.PXEVlanID
	pxeSVI := iface{Name: fmt.Sprintf("vlan%d", pxe), VLANID: pxe, Address: cfg.MetalCoreCIDR}
	netdev(41, "svi", pxeSVI)
	vxlan(100000+uint32(pxe), pxe, false)
	bridge := iface{Name: "bridge", SVIs: []string{pxeSVI.Name}, VLANs: []string{fmt.Sprint(pxe)}}

	for _, name := range slices.Sorted(maps.Keys(cfg.Ports.Vrfs)) {
		vrf := cfg.Ports.Vrfs[name]
		netdev(30, "vrf", iface{Name: name, VNI: vrf.VNI})

		svi := iface{Name: fmt.Sprintf("vlan%d", vrf.VLANID), VLANID: vrf.VLANID, VRF: name}
		netdev(41, "svi", svi)
		vxlan(vrf.VNI, vrf.VLANID, true)
		bridge.SVIs = append(bridge.SVIs, svi.Name)
		bridge.VLANs = append(bridge.VLANs, fmt.Sprint(vrf.VLANID))

		for _, port := range vrf.Neighbors {
			add(50, "vrf-port.network", iface{Name: port, VRF: name, Down: down(port)})
		}
	}

	bridge.VLANs = append(bridge.VLANs, cfg.AdditionalBridgeVIDs...)
	netdev(40, "bridge", bridge)

	for _, port := range cfg.Ports.Unprovisioned {
		add(51, "unprovisioned.network", iface{Name: port, VLANID: pxe, Down: down(port)})
	}

	return files, errors.Join(errs...)
}

// cleanup deletes netdevs whose .netdev file is removed or changed (networkd
// creates the changed ones again on reload) and releases ports that no
// .network file matches anymore from their bridge or VRF.
func (a *networkdApplier) cleanup(current, desired map[string][]byte) error {
	netdevs := map[string]bool{}
	for name := range current {
		if iface, kind := splitName(name); kind == "netdev" {
			netdevs[iface] = true
		}
	}
	configured := map[string]bool{}
	for name := range desired {
		if iface, kind := splitName(name); kind == "network" {
			configured[iface] = true
		}
	}

	var errs []error
	for _, name := range slices.Sorted(maps.Keys(current)) {
		iface, kind := splitName(name)
		switch {
		case kind == "netdev" && !bytes.Equal(current[name], desired[name]):
			a.log.Info("delete netdev", "netdev", iface, "file", name)
			if err := a.links.Delete(iface); err != nil {
				errs = append(errs, fmt.Errorf("failed to delete netdev %s: %w", iface, err))
			}
		case kind == "network" && !netdevs[iface] && !configured[iface]:
			a.log.Info("release link", "link", iface, "file", name)
			if err := a.links.Release(iface); err != nil {
				errs = append(errs, fmt.Errorf("failed to release link %s: %w", iface, err))
			}
		}
	}

	return errors.Join(errs...)
}

// splitName returns the interface and the kind (netdev or network) of a
// managed file.
func splitName(file string) (iface, kind string) {
	m := managedFile.FindStringSubmatch(file)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

func readManagedFiles(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{}
	for _, e := range entries {
		if !e.Type().IsRegular() || !managedFile.MatchString(e.Name()) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = content
	}
	return files, nil
}

// writeFiles writes the desired files that changed and removes the current
// ones that are no longer desired. Both maps only hold managed files.
func writeFiles(dir string, current, desired map[string][]byte) error {
	for name, content := range desired {
		if bytes.Equal(current[name], content) {
			continue
		}
		tmp := filepath.Join(dir, "."+name+".tmp")
		if err := os.WriteFile(tmp, content, 0644); err != nil { // nolint:gosec
			return err
		}
		if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	for name := range current {
		if _, ok := desired[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (netlinkLinks) Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return err
	}
	return netlink.LinkDel(link)
}

func (netlinkLinks) Release(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return err
	}
	if link.Attrs().MasterIndex == 0 {
		return nil
	}
	return netlink.LinkSetNoMaster(link)
}
