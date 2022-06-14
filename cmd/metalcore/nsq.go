package metalcore

import (
	"strings"
	"time"

	"github.com/metal-stack/go-hal/pkg/api"
	metalgo "github.com/metal-stack/metal-go"

	"github.com/metal-stack/metal-core/internal/event"
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

func (s *Server) timeoutHandler(err bus.TimeoutError) error {
	s.Log.Sugar().Errorw("timeout processing event", "event", err.Event())
	return nil
}

func (s *Server) initConsumer() error {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     s.Config.MQCACertFile,
		ClientCertFile: s.Config.MQClientCertFile,
	}
	c, err := bus.NewConsumer(s.Log, tlsCfg, s.Config.MQAddress)
	if err != nil {
		return err
	}

	evh := event.NewHandler(s.Log)

	err = c.With(bus.LogLevel(mapLogLevel(s.Config.MQLogLevel))).
		MustRegister(s.Config.MachineTopic, "core").
		Consume(event.MachineEvent{}, func(message interface{}) error {
			evt := message.(*event.MachineEvent)
			s.Log.Sugar().Debugw("got message", "topic", s.Config.MachineTopic, "channel", "core", "event", evt)
			switch evt.Type {
			case event.Delete:
				evh.FreeMachine(*evt)
			case event.Command:
				switch evt.Cmd.Command {
				case event.MachineOnCmd:
					evh.PowerOnMachine(*evt)
				case event.MachineOffCmd:
					evh.PowerOffMachine(*evt)
				case event.MachineResetCmd:
					evh.PowerResetMachine(*evt)
				case event.MachineCycleCmd:
					evh.PowerCycleMachine(*evt)
				case event.MachineBiosCmd:
					evh.PowerBootBiosMachine(*evt)
				case event.MachineDiskCmd:
					evh.PowerBootDiskMachine(*evt)
				case event.MachinePxeCmd:
					evh.PowerBootPxeMachine(*evt)
				case event.MachineReinstallCmd:
					evh.ReinstallMachine(*evt)
				case event.ChassisIdentifyLEDOnCmd:
					evh.PowerOnChassisIdentifyLED(*evt)
				case event.ChassisIdentifyLEDOffCmd:
					evh.PowerOffChassisIdentifyLED(*evt)
				case event.UpdateFirmwareCmd:
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
						go evh.UpdateBios(revision, description, s3Cfg, *evt)
					case metalgo.Bmc:
						go evh.UpdateBmc(revision, description, s3Cfg, *evt)
					default:
						s.Log.Sugar().Warnw("unknown firmware kind",
							"topic", s.Config.MachineTopic,
							"channel", "core",
							"firmware kind", string(kind),
							"event", evt,
						)
					}
				default:
					s.Log.Sugar().Warnw("unhandled command",
						"topic", s.Config.MachineTopic,
						"channel", "core",
						"event", evt,
					)
				}
			case event.Create, event.Update:
				fallthrough
			default:
				s.Log.Sugar().Warn("unhandled event",
					"topic", s.Config.MachineTopic,
					"channel", "core",
					"event", evt,
				)
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, s.timeoutHandler), bus.TTL(time.Duration(s.Config.MachineTopicTTL)*time.Millisecond))

	return err
}
