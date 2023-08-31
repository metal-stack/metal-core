package switcher

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
)

type NOS interface {
	SanitizeConfig(cfg *types.Conf)
	Apply(cfg *types.Conf) error
	IsInitialized() (initialized bool, err error)
	GetNics(log *slog.Logger, blacklist []string) ([]*models.V1SwitchNic, error)
	GetSwitchPorts() ([]*net.Interface, error)
	GetOS() (*models.V1SwitchOS, error)
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
