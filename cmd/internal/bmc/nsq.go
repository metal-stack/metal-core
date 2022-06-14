package bmc

import (
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
	b.log.Sugar().Errorw("timeout processing event", "event", err.Event())
	return nil
}

func (b *BMCService) InitConsumer() error {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     b.mqCACertFile,
		ClientCertFile: b.mqClientCertFile,
	}
	c, err := bus.NewConsumer(b.log, tlsCfg, b.mqAddress)
	if err != nil {
		return err
	}

	err = c.With(bus.LogLevel(mapLogLevel(b.mqLogLevel))).
		MustRegister(b.machineTopic, "core").
		Consume(MachineEvent{}, func(message interface{}) error {
			evt := message.(*MachineEvent)
			b.log.Sugar().Debugw("got message", "topic", b.machineTopic, "channel", "core", "event", evt)
			switch evt.Type {
			case Delete:
				b.FreeMachine(evt)
			case Command:
				switch evt.Cmd.Command {
				case MachineOnCmd:
					b.PowerOnMachine(evt)
				case MachineOffCmd:
					b.PowerOffMachine(evt)
				case MachineResetCmd:
					b.PowerResetMachine(evt)
				case MachineCycleCmd:
					b.PowerCycleMachine(evt)
				case MachineBiosCmd:
					b.PowerBootBiosMachine(evt)
				case MachineDiskCmd:
					b.PowerBootDiskMachine(evt)
				case MachinePxeCmd:
					b.PowerBootPxeMachine(evt)
				case MachineReinstallCmd:
					b.ReinstallMachine(evt)
				case ChassisIdentifyLEDOnCmd:
					b.PowerOnChassisIdentifyLED(evt)
				case ChassisIdentifyLEDOffCmd:
					b.PowerOffChassisIdentifyLED(evt)
				case UpdateFirmwareCmd:
					kind := metalgo.FirmwareKind(evt.Cmd.Params[0])
					revision := evt.Cmd.Params[1]
					description := evt.Cmd.Params[2]
					s3Cfg := &api.S3Config{
						Url:            evt.Cmd.Params[3],
						Key:            evt.Cmd.Params[4],
						Secret:         evt.Cmd.Params[5],
						FirmwareBucket: evt.Cmd.Params[6],
					}
					switch kind {
					case metalgo.Bios:
						go b.UpdateBios(revision, description, s3Cfg, evt)
					case metalgo.Bmc:
						go b.UpdateBmc(revision, description, s3Cfg, evt)
					default:
						b.log.Sugar().Warnw("unknown firmware kind",
							"topic", b.machineTopic,
							"channel", "core",
							"firmware kind", string(kind),
							"event", evt,
						)
					}
				default:
					b.log.Sugar().Warnw("unhandled command",
						"topic", b.machineTopic,
						"channel", "core",
						"event", evt,
					)
				}
			case Create, Update:
				fallthrough
			default:
				b.log.Sugar().Warn("unhandled event",
					"topic", b.machineTopic,
					"channel", "core",
					"event", evt,
				)
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, b.timeoutHandler), bus.TTL(time.Duration(b.machineTopicTTL)*time.Millisecond))

	return err
}
