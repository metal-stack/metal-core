package metalcore

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/client/partition"
	sw "github.com/metal-stack/metal-core/client/switch_operations"
	"github.com/metal-stack/metal-core/internal/api"
	"github.com/metal-stack/metal-core/internal/core"
	"github.com/metal-stack/metal-core/internal/endpoint"
	"github.com/metal-stack/metal-core/internal/event"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/metal-stack/v"
	"go.uber.org/zap"
)

func Create() *Server {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		panic(fmt.Errorf("Bad configuration:\n%+v", cfg))
	}
	os.Setenv(zapup.KeyFieldApp, "Metal-Core")
	os.Setenv(zapup.KeyLogLevel, cfg.LogLevel)
	if cfg.ConsoleLogging {
		os.Setenv(zapup.KeyLogEncoding, "console")
	}

	zapup.MustRootLogger().Info("Metal-Core Version",
		zap.Any("version", v.V),
	)

	devMode := strings.Contains(cfg.PartitionID, "vagrant")

	zapup.MustRootLogger().Info("Configuration",
		zap.Bool("DevMode", devMode),
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
		zap.Int("gRPC-port", cfg.GrpcPort),
		zap.String("gRPC-CACertFile", cfg.GrpcCACertFile),
		zap.String("gRPC-clientCertFile", cfg.GrpcClientCertFile),
		zap.String("gRPC-clientKeyFile", cfg.GrpcClientKeyFile),
	)

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), cfg.ApiBasePath, []string{cfg.ApiProtocol})

	app := &Server{
		AppContext: &domain.AppContext{
			Config:          cfg,
			MachineClient:   machine.New(transport, strfmt.Default),
			PartitionClient: partition.New(transport, strfmt.Default),
			SwitchClient:    sw.New(transport, strfmt.Default),
			DevMode:         devMode,
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)
	app.SetEndpointHandler(endpoint.NewHandler)
	app.SetEventHandler(event.NewHandler)
	app.InitHMAC()

	err := app.initConsumer()
	if err != nil {
		zapup.MustRootLogger().Fatal("failed to init NSQ consumer",
			zap.Error(err),
		)
		os.Exit(1)
	}

	s, err := app.APIClient().RegisterSwitch()
	if err != nil {
		zapup.MustRootLogger().Fatal("failed to register switch",
			zap.Error(err),
		)
		os.Exit(1)
	}

	app.initSwitchReconfiguration()
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
