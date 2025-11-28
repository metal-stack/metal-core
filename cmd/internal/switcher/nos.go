package switcher

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type NOS interface {
	SanitizeConfig(cfg *types.Conf)
	Apply(cfg *types.Conf) error
	IsInitialized() (initialized bool, err error)
	GetNics(log *slog.Logger, blacklist []string) ([]*apiv2.SwitchNic, error)
	GetSwitchPorts() ([]*net.Interface, error)
	GetOS() (*apiv2.SwitchOS, error)
	GetManagement() (ip, user string, err error)
}

func NewNOS(log *slog.Logger, frrTplFile, interfacesTplFile string) (NOS, error) {
	if _, err := os.Stat(sonic.SonicVersionFile); err == nil {
		log.Info("create sonic NOS")
		nos, err := sonic.New(log.With("os", "sonic"), frrTplFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SONiC NOS %w", err)
		}
		return nos, nil
	}
	log.Info("create cumulus NOS")
	return cumulus.New(log.With("os", "cumulus"), frrTplFile, interfacesTplFile), nil
}
