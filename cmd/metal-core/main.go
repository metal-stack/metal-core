package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	gonet "net"
	"os"
	"strings"
	"time"

	"github.com/emicklei/go-restful-openapi"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
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
		zap.String("LogLevel", cfg.LogLevel),
		zap.String("BindAddress", cfg.BindAddress),
		zap.String("IP", cfg.IP),
		zap.Int("Port", cfg.Port),
		zap.String("API-Protocol", cfg.ApiProtocol),
		zap.String("API-IP", cfg.ApiIP),
		zap.Int("API-Port", cfg.ApiPort),
	)

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	app := &app{
		AppContext: &domain.AppContext{
			Config:       cfg,
			DeviceClient: device.New(transport, strfmt.Default),
			SwitchClient: sw.New(transport, strfmt.Default),
		},
	}
	app.SetAPIClient(api.Client)
	app.SetServer(core.Server)
	app.SetEndpointHandler(endpoint.Handler)
	app.SetEventHandler(event.Handler)

	app.initConsumer()

	s, err := app.registerSwitch()

	if err != nil {
		zapup.MustRootLogger().Fatal("unable to register",
			zap.Error(err),
		)
		os.Exit(1)
	}

	app.BootConfig = &domain.BootConfig{
		MetalHammerImageURL:    *s.Site.Bootconfig.Imageurl,
		MetalHammerKernelURL:   *s.Site.Bootconfig.Kernelurl,
		MetalHammerCommandLine: *s.Site.Bootconfig.Commandline,
	}

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	return app
}

func (a *app) initConsumer() {
	_ = bus.NewConsumer(zapup.MustRootLogger(), a.Config.MQAddress).
		MustRegister("device", "rack1").
		Consume(domain.DeviceEvent{}, func(message interface{}) error {
			evt := message.(*domain.DeviceEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.Any("event", evt),
			)
			if evt.Type == domain.Delete {
				a.EventHandler().FreeDevice(evt.Old)
			}
			return nil
		}, 5)
}

func (a *app) registerSwitch() (*models.MetalSwitch, error) {
	var err error
	var nics []*models.MetalNic
	var hostname string

	if nics, err = getNics(); err != nil {
		return nil, fmt.Errorf("unable to get nics:%v", err)
	}

	if hostname, err = os.Hostname(); err != nil {
		return nil, fmt.Errorf("unable to get hostname:%v", err)
	}

	params := sw.NewRegisterSwitchParams()
	params.Body = &models.MetalRegisterSwitch{
		ID:     &hostname,
		SiteID: &a.Config.SiteID,
		RackID: &a.Config.RackID,
		Nics:   nics,
	}

	for {
		if ok, created, err := a.SwitchClient.RegisterSwitch(params); err == nil {
			if ok != nil {
				return ok.Payload, nil
			}
			return created.Payload, nil
		}
		zapup.MustRootLogger().Error("unable to register at metal-api",
			zap.Error(err),
		)
		time.Sleep(time.Second)
	}
}

func getNics() ([]*models.MetalNic, error) {
	var nics []*models.MetalNic
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("unable to get all links:%v", err)
	}
	for _, l := range links {
		attrs := l.Attrs()
		name := attrs.Name
		mac := attrs.HardwareAddr.String()
		_, err := gonet.ParseMAC(mac)
		if err != nil {
			zapup.MustRootLogger().Info("skip interface with invalid mac",
				zap.String("interface", name),
				zap.String("mac", mac),
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
	cfg := core.Init(endpoint.Handler(nil))
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, js, 0644); err != nil {
		fmt.Printf("%s\n", js)
	}
}
