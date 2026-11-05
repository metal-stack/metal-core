// Package bisdn is the switcher for BISDN Linux (https://www.bisdn.de): a
// Yocto-based NOS where baseboxd mirrors the Linux kernel's state into the
// ASIC via OF-DPA. Front panel ports are the netdevs port1..portN, the
// management port is enp0. metal-core renders systemd-networkd files for the
// kernel side and frr.conf for FRR, which runs natively (frr.service).
//
// Port breakout and FEC are not configured here: they live in the Broadcom
// SDK configuration (config.bcm, /etc/ofdpa/rc.soc), are set during ZTP and
// need an ofdpa restart, which metal-core never does.
package bisdn

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal"
	corenet "github.com/metal-stack/metal-core/cmd/internal/net"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	// OSReleaseFile identifies BISDN Linux by ID=bisdn-linux.
	OSReleaseFile = "/etc/os-release"
	osReleaseID   = "bisdn-linux"

	// DefaultManagementInterface is the name the BISDN installer gives the
	// management port (/lib/systemd/network/90-enp.link).
	DefaultManagementInterface = "enp0"
)

// frontPanel matches the front panel netdevs baseboxd creates.
var frontPanel = regexp.MustCompile(`^port[0-9]+$`)

type BISDN struct {
	frrApplier      *templates.Applier
	networkdApplier *networkdApplier
	mgmtInterface   string
	log             *slog.Logger
}

// IsBISDN reports whether the os-release file at path describes BISDN Linux.
func IsBISDN(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }() // read only
	return osRelease(f)["ID"] == osReleaseID
}

func New(log *slog.Logger, frrTplFile, networkdTplFile, mgmtInterface string) *BISDN {
	if mgmtInterface == "" {
		mgmtInterface = DefaultManagementInterface
	}
	return &BISDN{
		frrApplier:      NewFrrApplier(log, frrTplFile),
		networkdApplier: newNetworkdApplier(log, templates.BISDNNetworkdTemplate(networkdTplFile)),
		mgmtInterface:   mgmtInterface,
		log:             log,
	}
}

func (b *BISDN) Apply(ctx context.Context, cfg *types.Conf) error {
	if len(cfg.Ports.BladePorts) > 0 {
		return fmt.Errorf("blade ports are not supported on BISDN Linux: %v", cfg.Ports.BladePorts)
	}

	err := b.networkdApplier.Apply(ctx, cfg)
	if err != nil {
		return err
	}

	return b.frrApplier.Apply(ctx, cfg)
}

// IsInitialized reports whether baseboxd has created the front panel ports.
func (b *BISDN) IsInitialized(ctx context.Context) (initialized bool, err error) {
	ports, err := b.GetSwitchPorts(ctx)
	if err != nil {
		return false, err
	}
	return len(ports) > 0, nil
}

func (b *BISDN) GetNics(ctx context.Context, blacklist []string) (nics []*apiv2.SwitchNic, err error) {
	ifs, err := b.GetSwitchPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all ifs: %w", err)
	}

	for _, iface := range ifs {
		name := iface.Name
		if slices.Contains(blacklist, name) {
			b.log.Debug("skip interface, because it is contained in the blacklist", "interface", name, "blacklist", blacklist)
			continue
		}

		linkStatus, err := corenet.GetLinkStatus(name)
		if err != nil {
			b.log.Error("failed to get link status", "port", name, "status", linkStatus, "error", err)
		}

		// The identifier is what machines see as LLDP port ID: metal-core's
		// LLDP sender announces the interface name (locally assigned).
		nics = append(nics, &apiv2.SwitchNic{
			Identifier: name,
			Name:       name,
			State: &apiv2.NicState{
				Actual: linkStatus,
			},
		})
	}

	return nics, nil
}

// GetSwitchPorts returns the front panel ports, sorted by port number.
func (b *BISDN) GetSwitchPorts(_ context.Context) ([]*net.Interface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	switchPorts := make([]*net.Interface, 0, len(ifs))
	for i := range ifs {
		iface := &ifs[i]
		if !frontPanel.MatchString(iface.Name) {
			b.log.Debug("skip interface, because only port<N> interfaces are front panels", "interface", iface.Name)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	slices.SortStableFunc(switchPorts, func(x, y *net.Interface) int {
		return portNumber(x.Name) - portNumber(y.Name)
	})
	return switchPorts, nil
}

func portNumber(name string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(name, "port"))
	return n
}

func (b *BISDN) SanitizeConfig(cfg *types.Conf) {
	// same VRF names as on SONiC, which bisdn_frr.tpl relies on
	cfg.CapitalizeVrfName()
	cfg.Ports.Eth0.Name = b.mgmtInterface
}

func (b *BISDN) GetOS() (*apiv2.SwitchOS, error) {
	f, err := os.Open(OSReleaseFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", OSReleaseFile, err)
	}
	defer func() { _ = f.Close() }() // read only

	version := osRelease(f)["VERSION_ID"]
	if version == "" {
		version = "unknown"
	}
	return &apiv2.SwitchOS{
		Vendor:  apiv2.SwitchOSVendor_SWITCH_OS_VENDOR_BISDN,
		Version: version,
	}, nil
}

func (b *BISDN) GetManagement() (ip, user string, err error) {
	ip, err = internal.GetManagementIP(b.mgmtInterface)
	if err != nil {
		return "", "", err
	}
	return ip, "root", nil
}

// osRelease parses os-release(5) key=value lines.
func osRelease(r io.Reader) map[string]string {
	values := map[string]string{}
	s := bufio.NewScanner(r)
	for s.Scan() {
		k, v, found := strings.Cut(strings.TrimSpace(s.Text()), "=")
		if !found || strings.HasPrefix(k, "#") {
			continue
		}
		values[k] = strings.Trim(v, `"'`)
	}
	return values
}
