package bmc

import (
	"fmt"
	"strings"
	"time"

	"github.com/metal-stack/go-hal/pkg/api"
	metalgo "github.com/metal-stack/metal-go"

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

			if event.IPMI == nil {
				b.log.Errorw("event does not contain ipmi details", "event", event)
				return fmt.Errorf("event does not contain ipmi details:%v", event)
			}
			outBand, err := b.outBand(event.IPMI)
			if err != nil {
				b.log.Errorw("power boot disk", "error", err)
				return err
			}

			switch event.Type {
			case Delete:
				b.FreeMachine(outBand)
			case Command:
				switch event.Cmd.Command {
				case MachineOnCmd:
					b.PowerOnMachine(outBand)
				case MachineOffCmd:
					b.PowerOffMachine(outBand)
				case MachineResetCmd:
					b.PowerResetMachine(outBand)
				case MachineCycleCmd:
					b.PowerCycleMachine(outBand)
				case MachineBiosCmd:
					b.PowerBootBiosMachine(outBand)
				case MachineDiskCmd:
					b.PowerBootDiskMachine(outBand)
				case MachinePxeCmd:
					b.PowerBootPxeMachine(outBand)
				case MachineReinstallCmd:
					b.ReinstallMachine(outBand)
				case ChassisIdentifyLEDOnCmd:
					b.PowerOnChassisIdentifyLED(outBand)
				case ChassisIdentifyLEDOffCmd:
					b.PowerOffChassisIdentifyLED(outBand)
				case UpdateFirmwareCmd:
					kind := metalgo.FirmwareKind(event.Cmd.Params[0])
					revision := event.Cmd.Params[1]
					description := event.Cmd.Params[2]
					s3Cfg := &api.S3Config{
						Url:            event.Cmd.Params[3],
						Key:            event.Cmd.Params[4],
						Secret:         event.Cmd.Params[5],
						FirmwareBucket: event.Cmd.Params[6],
					}
					switch kind {
					case metalgo.Bios:
						go b.UpdateBios(revision, description, s3Cfg, event, outBand)
					case metalgo.Bmc:
						go b.UpdateBmc(revision, description, s3Cfg, event, outBand)
					default:
						b.log.Warnw("unknown firmware kind",
							"topic", b.machineTopic,
							"channel", "core",
							"firmware kind", string(kind),
							"event", event,
						)
					}
				default:
					b.log.Warnw("unhandled command",
						"topic", b.machineTopic,
						"channel", "core",
						"event", event,
					)
				}
			case Create, Update:
				fallthrough
			default:
				b.log.Warnw("unhandled event",
					"topic", b.machineTopic,
					"channel", "core",
					"event", event,
				)
			}
			return nil
			// FIXME machineTopicTTL should be configured as Duration in config.go
		}, 5, bus.Timeout(receiverHandlerTimeout, b.timeoutHandler), bus.TTL(time.Duration(b.machineTopicTTL)*time.Millisecond))

	return err
}
