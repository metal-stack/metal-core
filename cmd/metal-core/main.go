package main

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful-openapi"
	"io/ioutil"
	gonet "net"
	"os"
	"strings"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/server"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/go-openapi/strfmt"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/jaypipes/ghw"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

var (
	appContext *domain.AppContext
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spec" {
		filename := ""
		if len(os.Args) > 2 {
			filename = os.Args[2]
		}
		buildSpec(filename)
	} else {
		prepare()
		appContext.Server().Run()
	}
}

func prepare() {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		zapup.MustRootLogger().Fatal("Bad configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}

	os.Setenv(zapup.KeyFieldApp, "Metal-Core")
	if cfg.ConsoleLogging {
		os.Setenv(zapup.KeyLogEncoding, "console")
	}

	zapup.MustRootLogger().Info("Metal-Core Version",
		zap.Any("version", version.V),
	)

	zapup.MustRootLogger().Info("Configuration",
		zap.String("LogLevel", cfg.LogLevel),
		zap.String("BindAddress", cfg.BindAddress),
		zap.String("IP", cfg.IP),
		zap.Int("Port", cfg.Port),
		zap.String("API-Protocol", cfg.ApiProtocol),
		zap.String("API-IP", cfg.ApiIP),
		zap.Int("API-Port", cfg.ApiPort),
		zap.String("HammerImagePrefix", cfg.HammerImagePrefix),
	)

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	appContext = &domain.AppContext{
		Config:              cfg,
		ApiClientHandler:    api.Handler,
		ServerHandler:       server.Handler,
		EndpointHandler:     endpoint.Handler,
		EventHandlerHandler: event.Handler,
		DeviceClient:        device.New(transport, strfmt.Default),
		SwitchClient:        sw.New(transport, strfmt.Default),
	}

	initConsumer()

	registerSwitch()

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		os.Setenv("DEBUG", "1")
	}
}

func initConsumer() {
	bus.NewConsumer(zapup.MustRootLogger(), appContext.Config.MQAddress).
		MustRegister("device", "rack1").
		Consume(domain.DeviceEvent{}, func(message interface{}) error {
			evt := message.(*domain.DeviceEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.Any("event", evt),
			)
			if evt.Type == domain.DELETE {
				appContext.EventHandler().FreeDevice(evt.Old)
			}
			return nil
		}, 5)
}

func registerSwitch() {
	nics, err := getNics()
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to determine network interfaces",
			zap.Error(err),
		)
		os.Exit(1)
	}

	hostname, err := os.Hostname()
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to determine hostname",
			zap.Error(err),
		)
		os.Exit(1)
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.MetalSwitch{
		ID:     &hostname,
		SiteID: &appContext.Config.SiteID,
		RackID: &appContext.Config.RackID,
		Nics:   nics,
	}

	_, _, err = appContext.SwitchClient.RegisterSwitch(params)
	if err != nil {
		zapup.MustRootLogger().Fatal("unable to register at metal-api",
			zap.Error(err),
		)
		os.Exit(1)
	}
}

func getNics() ([]*models.MetalNic, error) {
	net, err := ghw.Network()
	if err != nil {
		return nil, fmt.Errorf("unable to get system nic(s), info:%v", err)
	}
	nics := []*models.MetalNic{}
	for _, n := range net.NICs {
		_, err := gonet.ParseMAC(n.MacAddress)
		if err != nil {
			zapup.MustRootLogger().Info("skip interface with invalid mac",
				zap.String("interface", n.Name),
				zap.String("mac", n.MacAddress),
			)
			continue
		}
		nic := &models.MetalNic{
			Mac:  &n.MacAddress,
			Name: &n.Name,
		}
		nics = append(nics, nic)
	}
	return nics, nil
}

func buildSpec(filename string) {
	cfg := server.Init(endpoint.Handler(nil))
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, js, 0644); err != nil {
		fmt.Printf("%s\n", js)
	}
}
