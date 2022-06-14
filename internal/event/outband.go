package event

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/connect"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"

	"go.uber.org/zap"
)

func outBand(ipmi IPMI, log *zap.SugaredLogger) (hal.OutBand, error) {
	host, portString, found := strings.Cut(ipmi.Address, ":")
	if !found {
		portString = "623"

	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("unable to convert port to an int %w", err)
	}
	outBand, err := connect.OutBand(host, port, ipmi.User, ipmi.Password, halzap.New(log))
	if err != nil {
		return nil, err
	}
	return outBand, nil
}
