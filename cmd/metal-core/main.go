package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/partition"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/go-openapi/strfmt"

	"github.com/metal-pod/v"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

// timeout for the nsq handler methods
const receiverHandlerTimeout = 15 * time.Second

type app struct {
	*domain.AppContext
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spec" {
		filename := ""
		if len(os.Args) > 2 {
			filename = os.Args[2]
		}
		buildSpec(filename)
	} else {
		app := prepare()
		app.Server().Run()
	}
}

func prepare() *app {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		zapup.MustRootLogger().Fatal("Bad configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}

	_ = os.Setenv(zapup.KeyFieldApp, "Metal-Core")
	if cfg.ConsoleLogging {
		_ = os.Setenv(zapup.KeyLogEncoding, "console")
	}

	zapup.MustRootLogger().Info("Metal-Core Version",
		zap.Any("version", v.V),
	)

	zapup.MustRootLogger().Info("Configuration",
		zap.String("CIDR", cfg.CIDR),
		zap.String("PartitionID", cfg.PartitionID),
		zap.String("RackID", cfg.RackID),
		zap.String("BindAddress", cfg.BindAddress),
		zap.Int("Port", cfg.Port),
		zap.String("LogLevel", cfg.LogLevel),
		zap.Bool("ConsoleLogging", cfg.ConsoleLogging),
		zap.String("API-Protocol", cfg.ApiProtocol),
		zap.String("API-IP", cfg.ApiIP),
		zap.Int("API-Port", cfg.ApiPort),
		zap.String("API-BasePath", cfg.ApiBasePath),
		zap.String("MQAddress", cfg.MQAddress),
		zap.String("MQLogLevel", cfg.MQLogLevel),
		zap.String("MachineTopic", cfg.MachineTopic),
		zap.String("LoopbackIP", cfg.LoopbackIP),
		zap.String("ASN", cfg.ASN),
		zap.String("SpineUplinks", cfg.SpineUplinks),
		zap.Bool("ReconfigureSwitch", cfg.ReconfigureSwitch),
		zap.String("ReconfigureSwitchInterval", cfg.ReconfigureSwitchInterval.String()),
		zap.String("ManagementGateway", cfg.ManagementGateway),
		zap.Any("AdditionalBridgeVIDs", cfg.AdditionalBridgeVIDs),
		zap.Any("AdditionalBridgePorts", cfg.AdditionalBridgePorts),
	)

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), cfg.ApiBasePath, []string{cfg.ApiProtocol})

	app := &app{
		AppContext: &domain.AppContext{
			Config:          cfg,
			MachineClient:   machine.New(transport, strfmt.Default),
			PartitionClient: partition.New(transport, strfmt.Default),
			SwitchClient:    sw.New(transport, strfmt.Default),
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)
	app.SetEndpointHandler(endpoint.NewHandler)
	app.SetEventHandler(event.NewHandler)
	app.InitHMAC()

	app.initConsumer()

	s, err := app.APIClient().RegisterSwitch()
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to register",
			zap.Error(err),
		)
		os.Exit(1)
	}

	host, err := os.Hostname()
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to detect hostname",
			zap.Error(err),
		)
		os.Exit(1)
	}
	err = app.EventHandler().ReconfigureSwitch(host)
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to fetch and apply current switch configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}

	app.APIClient().ConstantlyPhoneHome()

	app.BootConfig = &domain.BootConfig{
		MetalHammerImageURL:    s.Partition.Bootconfig.Imageurl,
		MetalHammerKernelURL:   s.Partition.Bootconfig.Kernelurl,
		MetalHammerCommandLine: s.Partition.Bootconfig.Commandline,
	}

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	return app
}

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

func (a *app) initConsumer() {
	_ = bus.NewTLSConsumer(zapup.MustRootLogger(), a.Config.CACertFile, a.Config.ClientCertFile, a.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(a.Config.MQLogLevel))).
		MustRegister(a.Config.MachineTopic, "core").
		Consume(domain.MachineEvent{}, func(message interface{}) error {
			evt := message.(*domain.MachineEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.String("topic", a.Config.MachineTopic),
				zap.String("channel", "core"),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Delete:
				a.EventHandler().FreeMachine(*evt.Old.ID)
			case domain.Command:
				switch evt.Cmd.Command {
				case domain.MachineOnCmd:
					a.EventHandler().PowerOnMachine(*evt.Cmd.Target.ID)
				case domain.MachineOffCmd:
					a.EventHandler().PowerOffMachine(*evt.Cmd.Target.ID)
				case domain.MachineResetCmd:
					a.EventHandler().PowerResetMachine(*evt.Cmd.Target.ID)
				case domain.MachineBiosCmd:
					a.EventHandler().BootBiosMachine(*evt.Cmd.Target.ID)
				case domain.ChassisIdentifyLEDOnCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					a.EventHandler().PowerOnChassisIdentifyLED(*evt.Cmd.Target.ID, description)
				case domain.ChassisIdentifyLEDOffCmd:
					description := strings.TrimSpace(strings.Join(evt.Cmd.Params, " "))
					if len(description) == 0 {
						description = "unknown"
					}
					a.EventHandler().PowerOffChassisIdentifyLED(*evt.Cmd.Target.ID, description)
				default:
					zapup.MustRootLogger().Warn("Unhandled command",
						zap.String("topic", a.Config.MachineTopic),
						zap.String("channel", "core"),
						zap.Any("event", evt),
					)
				}
			default:
				zapup.MustRootLogger().Warn("Unhandled event",
					zap.String("topic", a.Config.MachineTopic),
					zap.String("channel", "core"),
					zap.Any("event", evt),
				)
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler))

	hostname, _ := os.Hostname()

	_ = bus.NewTLSConsumer(zapup.MustRootLogger(), a.Config.CACertFile, a.Config.ClientCertFile, a.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(a.Config.MQLogLevel))).
		// the hostname is used here as channel name
		// this is intended so that messages in the switch topics get replicated
		// to all channels leaf01, leaf02
		MustRegister(a.Config.SwitchTopic, hostname).
		Consume(domain.SwitchEvent{}, func(message interface{}) error {
			evt := message.(*domain.SwitchEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.String("topic", a.Config.SwitchTopic),
				zap.String("channel", hostname),
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Update:
				for _, s := range evt.Switches {
					sid := *s.ID
					if sid == hostname {
						err := a.EventHandler().ReconfigureSwitch(sid)
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
					zap.String("topic", a.Config.SwitchTopic),
					zap.String("channel", hostname),
					zap.Any("event", evt),
				)
			}
			return nil
		}, 1, bus.Timeout(receiverHandlerTimeout, timeoutHandler))
}

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Error("Timeout processing event", zap.Any("event", err.Event()))
	return nil
}

func buildSpec(filename string) {
	cfg := core.Init(endpoint.NewHandler(nil))
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, js, 0644); err != nil {
		fmt.Printf("%s\n", js)
	}
}
