package switcher

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/bisdn"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type NOS interface {
	SanitizeConfig(cfg *types.Conf)
	Apply(ctx context.Context, cfg *types.Conf) error
	IsInitialized(ctx context.Context) (initialized bool, err error)
	GetNics(ctx context.Context, blacklist []string) ([]*apiv2.SwitchNic, error)
	GetOS() (*apiv2.SwitchOS, error)
	GetSwitchPorts(ctx context.Context) ([]*net.Interface, error)
	GetManagement() (ip, user string, err error)
}

// ManagementInterface returns the configured management interface, or the
// running NOS's default if none is configured.
func ManagementInterface(configured string) string {
	if configured != "" {
		return configured
	}
	if bisdn.IsBISDN(bisdn.OSReleaseFile) {
		return bisdn.DefaultManagementInterface
	}
	return "eth0"
}

func NewNOS(log *slog.Logger, frrTplFile, interfacesTplFile string, interfaceNamingSchema sonic.InterfaceNamingSchema, mgmtInterface string) (NOS, error) {
	if bisdn.IsBISDN(bisdn.OSReleaseFile) {
		log.Info("create BISDN Linux NOS")
		// on BISDN Linux, the interfaces template renders the systemd-networkd files
		return bisdn.New(log.With("os", "bisdn"), frrTplFile, interfacesTplFile, mgmtInterface), nil
	}
	if _, err := os.Stat(sonic.SonicVersionFile); err == nil {
		log.Info("create sonic NOS")
		nos, err := sonic.New(log.With("os", "sonic"), frrTplFile, interfaceNamingSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SONiC NOS %w", err)
		}
		return nos, nil
	}
	log.Info("create cumulus NOS")
	return cumulus.New(log.With("os", "cumulus"), frrTplFile, interfacesTplFile), nil
}
