package metalcore

import (
	"strings"
	"time"

	"github.com/metal-stack/go-hal/pkg/api"
	"github.com/metal-stack/metal-core/pkg/domain"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-lib/bus"
	"go.uber.org/zap"
)

// timeout for the nsq handler methods
const receiverHandlerTimeout = 15 * time.Second

type mqClient struct {
	addr     string
	channel  string
	log      *zap.Logger
	logLevel bus.Level
	tlsCfg   *bus.TLSConfig
	topic    string
	ttl      time.Duration
}

func newMQClient(cfg *domain.Config, log *zap.Logger) *mqClient {
	tlsCfg := &bus.TLSConfig{
		CACertFile:     cfg.MQCACertFile,
		ClientCertFile: cfg.MQClientCertFile,
	}
	ttl := time.Duration(cfg.MachineTopicTTL) * time.Millisecond
	return &mqClient{
		addr:     cfg.MQAddress,
		channel:  "core",
		log:      log,
		logLevel: mapLogLevel(cfg.MQLogLevel),
		tlsCfg:   tlsCfg,
		ttl:      ttl,
	}
}

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

func (m *mqClient) initConsumer(handler domain.EventHandler) error {
	c, err := bus.NewConsumer(m.log, m.tlsCfg, m.addr)
	if err != nil {
		return err
	}

	subLogger := m.log.With(
		zap.String("topic", m.topic),
		zap.String("channel", m.channel),
	)
	logLevel := bus.LogLevel(m.logLevel)
	receiver := getReceiver(handler, subLogger)
	timeout := bus.Timeout(receiverHandlerTimeout, m.timeoutHandler)
	ttl := bus.TTL(m.ttl)

	err = c.With(logLevel).
		MustRegister(m.topic, m.channel).
		Consume(domain.MachineEvent{}, receiver, 5, timeout, ttl)

	return err
}

func (m *mqClient) timeoutHandler(err bus.TimeoutError) error {
	m.log.Error("timeout processing event", zap.Any("event", err.Event()))
	return nil
}

func getReceiver(handler domain.EventHandler, log *zap.Logger) bus.Receiver {
	return func(message interface{}) error {
		evt := message.(*domain.MachineEvent)
		log.Debug("got message",
			zap.Any("event", evt),
		)
		switch evt.Type {
		case domain.Delete:
			handler.FreeMachine(evt.OldMachineID)
		case domain.Command:
			switch evt.Cmd.Command {
			case domain.MachineOnCmd:
				handler.PowerOnMachine(evt.Cmd.TargetMachineID)
			case domain.MachineOffCmd:
				handler.PowerOffMachine(evt.Cmd.TargetMachineID)
			case domain.MachineResetCmd:
				handler.PowerResetMachine(evt.Cmd.TargetMachineID)
			case domain.MachineCycleCmd:
				handler.PowerCycleMachine(evt.Cmd.TargetMachineID)
			case domain.MachineBiosCmd:
				handler.PowerBootBiosMachine(evt.Cmd.TargetMachineID)
			case domain.MachineDiskCmd:
				handler.PowerBootDiskMachine(evt.Cmd.TargetMachineID)
			case domain.MachinePxeCmd:
				handler.PowerBootPxeMachine(evt.Cmd.TargetMachineID)
			case domain.MachineReinstallCmd:
				handler.ReinstallMachine(evt.Cmd.TargetMachineID)
			case domain.ChassisIdentifyLEDOnCmd:
				description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
				if len(description) == 0 {
					description = "unknown"
				}
				handler.PowerOnChassisIdentifyLED(evt.Cmd.TargetMachineID, description)
			case domain.ChassisIdentifyLEDOffCmd:
				description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
				if len(description) == 0 {
					description = "unknown"
				}
				handler.PowerOffChassisIdentifyLED(evt.Cmd.TargetMachineID, description)
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
					go handler.UpdateBios(evt.Cmd.TargetMachineID, revision, description, s3Cfg)
				case metalgo.Bmc:
					go handler.UpdateBmc(evt.Cmd.TargetMachineID, revision, description, s3Cfg)
				default:
					log.Warn("unknown firmware kind",
						zap.String("firmware kind", string(kind)),
						zap.Any("event", evt),
					)
				}
			default:
				log.Warn("unhandled command",
					zap.Any("event", evt),
				)
			}
		case domain.Create, domain.Update:
			fallthrough
		default:
			log.Warn("unhandled event",
				zap.Any("event", evt),
			)
		}
		return nil
	}
}
