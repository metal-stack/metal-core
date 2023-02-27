package switcher

import (
	"net"

	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-go/api/models"
)

type NOS interface {
	Apply(cfg *types.Conf) error
	GetSwitchPorts() ([]*net.Interface, error)
	GetOS() (*models.V1SwitchOS, error)
	GetManagement() (ip, user string, err error)
}

func NewNOS(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) NOS {
	return cumulus.New(log.Named("cumulus"), frrTplFile, interfacesTplFile)
}
