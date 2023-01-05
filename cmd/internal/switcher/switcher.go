package switcher

import (
	"fmt"
	"os"

	"github.com/metal-stack/metal-core/cmd/internal/core"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"go.uber.org/zap"
)

func NewNOS(log *zap.SugaredLogger, frrTplFile, interfacesTplFile string) (core.NOS, error) {
	if _, err := os.Stat(sonic.SonicVersionFile); err == nil {
		nos, err := sonic.New(log.Named("sonic"), frrTplFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SONiC NOS %w", err)
		}
		return nos, nil
	}
	return cumulus.New(log.Named("cumulus"), frrTplFile, interfacesTplFile), nil
}
