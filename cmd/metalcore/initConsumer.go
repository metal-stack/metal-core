package metalcore

import (
	"strings"
	"time"

	"github.com/metal-stack/go-hal/pkg/api"
	metalgo "github.com/metal-stack/metal-go"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
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
	s.Log.Error("timeout processing event", zap.Any("event", err.Event()))
	return nil
}

func (s *Server) initSwitchReconfiguration() {
	go s.EventHandler().ReconfigureSwitch()
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

	err = c.With(bus.LogLevel(mapLogLevel(s.Config.MQLogLevel))).
		MustRegister(s.Config.MachineTopic, "core").
		Consume(domain.MachineEvent{}, func(message interface{}) error {
			evt := message.(*domain.MachineEvent)
			s.Log.Debug("got message",
				zap.String("topic", s.Config.MachineTopic),
				zap.String("channel", "core"),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Delete:
				// MachineID should be taken from evt.Cmd.TargetMachineID
				s.EventHandler().FreeMachine(*evt)
			case domain.Command:
				switch evt.Cmd.Command {
				case domain.MachineOnCmd:
					s.EventHandler().PowerOnMachine(*evt)
				case domain.MachineOffCmd:
					s.EventHandler().PowerOffMachine(*evt)
				case domain.MachineResetCmd:
					s.EventHandler().PowerResetMachine(*evt)
				case domain.MachineCycleCmd:
					s.EventHandler().PowerCycleMachine(*evt)
				case domain.MachineBiosCmd:
					s.EventHandler().PowerBootBiosMachine(*evt)
				case domain.MachineDiskCmd:
					s.EventHandler().PowerBootDiskMachine(*evt)
				case domain.MachinePxeCmd:
					s.EventHandler().PowerBootPxeMachine(*evt)
				case domain.MachineReinstallCmd:
					s.EventHandler().ReinstallMachine(*evt)
				case domain.ChassisIdentifyLEDOnCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOnChassisIdentifyLED(*evt)
				case domain.ChassisIdentifyLEDOffCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOffChassisIdentifyLED(*evt)
				case domain.UpdateFirmwareCmd:
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
						go s.EventHandler().UpdateBios(revision, description, s3Cfg, *evt)
					case metalgo.Bmc:
						go s.EventHandler().UpdateBmc(revision, description, s3Cfg, *evt)
					default:
						s.Log.Warn("unknown firmware kind",
							zap.String("topic", s.Config.MachineTopic),
							zap.String("channel", "core"),
							zap.String("firmware kind", string(kind)),
							zap.Any("event", evt),
						)
					}
				default:
					s.Log.Warn("unhandled command",
						zap.String("topic", s.Config.MachineTopic),
						zap.String("channel", "core"),
						zap.Any("event", evt),
					)
				}
			case domain.Create, domain.Update:
				fallthrough
			default:
				s.Log.Warn("unhandled event",
					zap.String("topic", s.Config.MachineTopic),
					zap.String("channel", "core"),
					zap.Any("event", evt),
				)
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, s.timeoutHandler), bus.TTL(time.Duration(s.Config.MachineTopicTTL)*time.Millisecond))

	return err
}
