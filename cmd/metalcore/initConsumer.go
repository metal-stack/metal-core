package metalcore

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

// timeout for the nsq handler methods
const receiverHandlerTimeout = 15 * time.Second

func mapLogLevel(level string) bus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return bus.Debug
	case "info":
		return bus.Info
	case "warn":
		return bus.Warning
	case "error":
		return bus.Error
	default:
		return bus.Info
	}
}

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Error("Timeout processing event", zap.Any("event", err.Event()))
	return nil
}

func (s *Server) initConsumer() {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     s.Config.MQCACertFile,
		ClientCertFile: s.Config.MQClientCertFile,
	}
	_ = bus.NewConsumer(zapup.MustRootLogger(), tlsCfg, s.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(s.Config.MQLogLevel))).
		MustRegister(s.Config.MachineTopic, "core").
		Consume(domain.MachineEvent{}, func(message interface{}) error {
			evt := message.(*domain.MachineEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.String("topic", s.Config.MachineTopic),
				zap.String("channel", "core"),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Delete:
				s.EventHandler().FreeMachine(*evt.Old.ID)
			case domain.Command:
				switch evt.Cmd.Command {
				case domain.MachineOnCmd:
					s.EventHandler().PowerOnMachine(*evt.Cmd.Target.ID)
				case domain.MachineOffCmd:
					s.EventHandler().PowerOffMachine(*evt.Cmd.Target.ID)
				case domain.MachineResetCmd:
					s.EventHandler().PowerResetMachine(*evt.Cmd.Target.ID)
				case domain.MachineBiosCmd:
					s.EventHandler().BootBiosMachine(*evt.Cmd.Target.ID)
				case domain.ChassisIdentifyLEDOnCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOnChassisIdentifyLED(*evt.Cmd.Target.ID, description)
				case domain.ChassisIdentifyLEDOffCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOffChassisIdentifyLED(*evt.Cmd.Target.ID, description)
				default:
					zapup.MustRootLogger().Warn("Unhandled command",
						zap.String("topic", s.Config.MachineTopic),
						zap.String("channel", "core"),
						zap.Any("event", evt),
					)
				}
			default:
				zapup.MustRootLogger().Warn("Unhandled event",
					zap.String("topic", s.Config.MachineTopic),
					zap.String("channel", "core"),
					zap.Any("event", evt),
				)
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler))

	hostname, _ := os.Hostname()

	_ = bus.NewConsumer(zapup.MustRootLogger(), tlsCfg, s.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(s.Config.MQLogLevel))).
		// the hostname is used here as channel name
		// this is intended so that messages in the switch topics get replicated
		// to all channels leaf01, leaf02
		MustRegister(s.Config.SwitchTopic, hostname).
		Consume(domain.SwitchEvent{}, func(message interface{}) error {
			evt := message.(*domain.SwitchEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.String("topic", s.Config.SwitchTopic),
				zap.String("channel", hostname),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Update:
				for _, sw := range evt.Switches {
					sid := *sw.ID
					if sid == hostname {
						err := s.EventHandler().ReconfigureSwitch(sid)
						if err != nil {
							zapup.MustRootLogger().Error("could not fetch and apply switch configuration", zap.Error(err))
						}
						return nil
					}
				}
				zapup.MustRootLogger().Info("Skip event because it is not intended for this switch",
					zap.Any("Machine", evt.Machine),
					zap.Any("Switches", evt.Switches),
					zap.String("Hostname", hostname),
				)
			default:
				zapup.MustRootLogger().Warn("Unhandled event",
					zap.String("topic", s.Config.SwitchTopic),
					zap.String("channel", hostname),
					zap.Any("event", evt),
				)
			}
			return nil
		}, 1, bus.Timeout(receiverHandlerTimeout, timeoutHandler))
}