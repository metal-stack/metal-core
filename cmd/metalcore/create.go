package metalcore

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/partition"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/metal-pod/v"
	"go.uber.org/zap"
	"os"
	"strings"
)

func Create() *Server {
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
		zap.String("MQCACertFile", cfg.MQCACertFile),
		zap.String("MQClientCertFile", cfg.MQClientCertFile),
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

	app := &Server{
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