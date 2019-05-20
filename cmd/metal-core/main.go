package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	gonet "net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	restfulspec "github.com/emicklei/go-restful-openapi"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/go-openapi/strfmt"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/vishvananda/netlink"

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
		zap.Any("version", version.V),
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

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	app := &app{
		AppContext: &domain.AppContext{
			Config:        cfg,
			MachineClient: machine.New(transport, strfmt.Default),
			SwitchClient:  sw.New(transport, strfmt.Default),
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)
	app.SetEndpointHandler(endpoint.NewHandler)
	app.SetEventHandler(event.NewHandler)

	app.initConsumer()

	s, err := app.registerSwitch()
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

	app.BootConfig = &domain.BootConfig{
		MetalHammerImageURL:    *s.Partition.BootConfiguration.ImageURL,
		MetalHammerKernelURL:   *s.Partition.BootConfiguration.KernelURL,
		MetalHammerCommandLine: *s.Partition.BootConfiguration.CommandLine,
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
	hostname, _ := os.Hostname()
	_ = bus.NewConsumer(zapup.MustRootLogger(), a.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(a.Config.MQLogLevel))).
		MustRegister(a.Config.MachineTopic, "core").
		Consume(domain.MachineEvent{}, func(message interface{}) error {
			evt := message.(*domain.MachineEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.Any("event", evt),
			)
			switch evt.Type {
			case domain.Delete:
				a.EventHandler().FreeMachine(*evt.Old.ID)
			case domain.Command:
				switch evt.Cmd.Command {
				case domain.MachineOnCmd:
					a.EventHandler().PowerOnMachine(*evt.Cmd.Target.ID, evt.Cmd.Params)
				case domain.MachineOffCmd:
					a.EventHandler().PowerOffMachine(*evt.Cmd.Target.ID, evt.Cmd.Params)
				case domain.MachineResetCmd:
					a.EventHandler().PowerResetMachine(*evt.Cmd.Target.ID, evt.Cmd.Params)
				case domain.MachineBiosCmd:
					a.EventHandler().BootBiosMachine(*evt.Cmd.Target.ID, evt.Cmd.Params)
				}
			}
			return nil
		}, 5, bus.Timeout(receiverHandlerTimeout, timeoutHandler))

	_ = bus.NewConsumer(zapup.MustRootLogger(), a.Config.MQAddress).
		With(bus.LogLevel(mapLogLevel(a.Config.MQLogLevel))).
		// the hostname is used here as channel name
		// this is intended so that messages in the switch topics get replicated
		// to all channels leaf01, leaf02
		MustRegister(a.Config.SwitchTopic, hostname).
		Consume(domain.SwitchEvent{}, func(message interface{}) error {
			evt := message.(*domain.SwitchEvent)
			zapup.MustRootLogger().Info("Got message",
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
			}
			return nil
		}, 1, bus.Timeout(receiverHandlerTimeout, timeoutHandler))
}

func timeoutHandler(err bus.TimeoutError) error {
	zapup.MustRootLogger().Error("Timeout processing event", zap.Any("event", err.Event()))
	return nil
}

func (a *app) registerSwitch() (*models.MetalSwitch, error) {
	var err error
	var nics []*models.MetalNic
	var hostname string

	if nics, err = getNics(a.Config.AdditionalBridgePorts); err != nil {
		return nil, errors.Wrap(err, "unable to get nics")
	}

	if hostname, err = os.Hostname(); err != nil {
		return nil, errors.Wrap(err, "unable to get hostname")
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.MetalRegisterSwitch{
		ID:          &hostname,
		PartitionID: &a.Config.PartitionID,
		RackID:      &a.Config.RackID,
		Nics:        nics,
	}

	for {
		ok, created, err := a.SwitchClient.RegisterSwitch(params)
		if err == nil {
			if ok != nil {
				return ok.Payload, nil
			}
			return created.Payload, nil
		}
		zapup.MustRootLogger().Error("unable to register at metal-api", zap.Error(err))
		time.Sleep(time.Second)
	}
}

func getNics(blacklist []string) ([]*models.MetalNic, error) {
	var nics []*models.MetalNic
	links, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get all links")
	}
links:
	for _, l := range links {
		attrs := l.Attrs()
		name := attrs.Name
		mac := attrs.HardwareAddr.String()
		for _, b := range blacklist {
			if b == name {
				zapup.MustRootLogger().Info("skip interface, because it is contained in the blacklist",
					zap.String("interface", name),
					zap.Any("blacklist", blacklist),
				)
				break links
			}
		}
		if !strings.HasPrefix(name, "swp") {
			zapup.MustRootLogger().Info("skip interface, because only swp* switch ports are reported to metal-api",
				zap.String("interface", name),
				zap.String("MAC", mac),
			)
			continue
		}
		_, err := gonet.ParseMAC(mac)
		if err != nil {
			zapup.MustRootLogger().Info("skip interface with invalid mac",
				zap.String("interface", name),
				zap.String("MAC", mac),
			)
			continue
		}
		nic := &models.MetalNic{
			Mac:  &mac,
			Name: &name,
		}
		nics = append(nics, nic)
	}
	return nics, nil
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
