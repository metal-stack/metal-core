package metalcore

import (
	"github.com/metal-stack/go-hal/pkg/api"
	"strings"
	"time"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/metal-stack/metal-lib/zapup"
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

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Error("Timeout processing event", zap.Any("event", err.Event()))
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
	c, err := bus.NewConsumer(zapup.MustRootLogger(), tlsCfg, s.Config.MQAddress)
	if err != nil {
		return err
	}

	err = c.With(bus.LogLevel(mapLogLevel(s.Config.MQLogLevel))).
		MustRegister(s.Config.MachineTopic, "core").
		Consume(domain.MachineEvent{}, func(message interface{}) error {
			evt := message.(*domain.MachineEvent)
			zapup.MustRootLogger().Debug("Got message",
				zap.String("topic", s.Config.MachineTopic),
				zap.String("channel", "core"),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Delete:
				s.EventHandler().FreeMachine(evt.OldMachineID)
			case domain.Command:
				switch evt.Cmd.Command {
				case domain.MachineOnCmd:
					s.EventHandler().PowerOnMachine(evt.Cmd.TargetMachineID)
				case domain.MachineOffCmd:
					s.EventHandler().PowerOffMachine(evt.Cmd.TargetMachineID)
				case domain.MachineResetCmd:
					s.EventHandler().PowerResetMachine(evt.Cmd.TargetMachineID)
				case domain.MachineBiosCmd:
					s.EventHandler().PowerBootBiosMachine(evt.Cmd.TargetMachineID)
				case domain.MachineDiskCmd:
					s.EventHandler().PowerBootDiskMachine(evt.Cmd.TargetMachineID)
				case domain.MachinePxeCmd:
					s.EventHandler().PowerBootPxeMachine(evt.Cmd.TargetMachineID)
				case domain.MachineReinstallCmd:
					s.EventHandler().ReinstallMachine(evt.Cmd.TargetMachineID)
				case domain.ChassisIdentifyLEDOnCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOnChassisIdentifyLED(evt.Cmd.TargetMachineID, description)
				case domain.ChassisIdentifyLEDOffCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					s.EventHandler().PowerOffChassisIdentifyLED(evt.Cmd.TargetMachineID, description)
				case domain.UpdateFirmwareCmd:
					kind := evt.Cmd.Params[0]
					revision := evt.Cmd.Params[1]
					description := evt.Cmd.Params[2]
					s3Cfg := &api.S3Config{
						Url:    evt.Cmd.Params[3],
						Key:    evt.Cmd.Params[4],
						Secret: evt.Cmd.Params[5],
					}
					switch kind {
					case "bios":
						s.EventHandler().UpdateBios(evt.Cmd.TargetMachineID, revision, description, s3Cfg)
					case "bmc":
						s.EventHandler().UpdateBmc(evt.Cmd.TargetMachineID, revision, description, s3Cfg)
					default:
						zapup.MustRootLogger().Warn("Unknown firmware kind",
							zap.String("topic", s.Config.MachineTopic),
							zap.String("channel", "core"),
							zap.String("firmware kind", kind),
							zap.Any("event", evt),
						)
					}
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
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler), bus.TTL(time.Duration(s.Config.MachineTopicTTL)*time.Millisecond))

	return err
}
