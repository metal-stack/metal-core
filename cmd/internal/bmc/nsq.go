package bmc

import (
	"fmt"
	"strings"
	"time"

	"github.com/metal-stack/go-hal"

	"github.com/metal-stack/metal-lib/bus"
)

// timeout for the nsq handler methods
const receiverHandlerTimeout = 15 * time.Second

func mapLogLevel(level string) bus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return bus.Debug
	case "info":
		return bus.Info
	case "warn", "warning":
		return bus.Warning
	case "error":
		return bus.Error
	default:
		return bus.Warning
	}
}

func (b *BMCService) timeoutHandler(err bus.TimeoutError) error {
	b.log.Errorw("timeout processing event", "event", err.Event())
	return nil
}

func (b *BMCService) InitConsumer() error {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     b.mqCACertFile,
		ClientCertFile: b.mqClientCertFile,
	}
	c, err := bus.NewConsumer(b.log.Desugar(), tlsCfg, b.mqAddress)
	if err != nil {
		return err
	}

	err = c.With(bus.LogLevel(mapLogLevel(b.mqLogLevel))).
		MustRegister(b.machineTopic, "core").
		Consume(MachineEvent{}, func(message interface{}) error {
			event := message.(*MachineEvent)
			b.log.Debugw("got message", "topic", b.machineTopic, "channel", "core", "event", event)

			if event.Cmd.IPMI == nil {
				return fmt.Errorf("event does not contain ipmi details:%v", event)
			}
			outBand, err := b.outBand(event.Cmd.IPMI)
			if err != nil {
				b.log.Errorw("error creating outband connection", "error", err)
				return err
			}

			switch event.Type {
			case Delete:
				err := outBand.BootFrom(hal.BootTargetPXE)
				if err != nil {
					return err
				}
				return outBand.PowerCycle()
			case Command:
				switch event.Cmd.Command {
				case MachineOnCmd:
					return outBand.PowerOn()
				case MachineOffCmd:
					return outBand.PowerOff()
				case MachineResetCmd:
					return outBand.PowerReset()
				case MachineCycleCmd:
					return outBand.PowerCycle()
				case MachineBiosCmd:
					return outBand.BootFrom(hal.BootTargetBIOS)
				case MachineDiskCmd:
					return outBand.BootFrom(hal.BootTargetDisk)
				case MachinePxeCmd:
					return outBand.BootFrom(hal.BootTargetPXE)
				case MachineReinstallCmd:
					err := outBand.BootFrom(hal.BootTargetPXE)
					if err != nil {
						return err
					}
					return outBand.PowerReset()
				case ChassisIdentifyLEDOnCmd:
					return outBand.IdentifyLEDOn()
				case ChassisIdentifyLEDOffCmd:
					return outBand.IdentifyLEDOff()
				case UpdateFirmwareCmd:
					b.UpdateFirmware(outBand, event)
				default:
					b.log.Errorw("unhandled command", "topic", b.machineTopic, "channel", "core", "event", event)
				}
			case Create, Update:
				fallthrough
			default:
				b.log.Warnw("unhandled event", "topic", b.machineTopic, "channel", "core", "event", event)
			}
			return nil
			// FIXME machineTopicTTL should be configured as Duration in config.go
		}, 5, bus.Timeout(receiverHandlerTimeout, b.timeoutHandler), bus.TTL(time.Duration(b.machineTopicTTL)*time.Millisecond))

	return err
}
