package switcher

import (
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

type Sonic struct {
	bgpApplier *networkApplier
	log        *zap.SugaredLogger
}

func NewSonic(log *zap.SugaredLogger) *Sonic {
	return &Sonic{
		bgpApplier: newBgpApplier(),
		log:        log,
	}
}

func (s *Sonic) Apply(cfg *Conf) error {
	return s.bgpApplier.Apply(cfg)
}

func (s *Sonic) GetSwitchPorts() ([]*net.Interface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get all interfaces: %w", err)
	}

	switchPorts := make([]*net.Interface, 0, len(ifs))
	for i := range ifs {
		iface := &ifs[i]
		if !strings.HasPrefix(iface.Name, "Ethernet") {
			s.log.Debug("skip interface, because only Ethernet* interface are front panels",
				zap.String("interface", iface.Name),
				zap.String("MAC", iface.HardwareAddr.String()),
			)
			continue
		}
		switchPorts = append(switchPorts, iface)
	}
	return switchPorts, nil
}
