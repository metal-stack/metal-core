package switcher

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
)

type NOS interface {
	SanitizeConfig(cfg *types.Conf)
	Apply(cfg *types.Conf) error
	GetNics(log *zap.SugaredLogger, blacklist []string) ([]*models.V1SwitchNic, error)
	GetSwitchPorts() ([]*net.Interface, error)
	GetOS() (*models.V1SwitchOS, error)
	GetManagement() (ip, user string, err error)
}

func NewNOS(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) (NOS, error) {
	if _, err := os.Stat(sonic.SonicVersionFile); err == nil {
		log.Infow("create sonic NOS")
		nos, err := sonic.New(log.Named("sonic"), frrTplFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SONiC NOS %w", err)
		}
		return nos, nil
	}
	log.Infow("create cumulus NOS")
	return cumulus.New(log.Named("cumulus"), frrTplFile, interfacesTplFile), nil
}
