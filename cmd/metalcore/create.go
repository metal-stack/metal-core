package metalcore

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/metal-stack/metal-core/internal/api"
	"github.com/metal-stack/metal-core/internal/core"
	"github.com/metal-stack/metal-core/internal/endpoint"
	"github.com/metal-stack/metal-core/internal/event"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/client/partition"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/v"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Create() *Server {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		panic(fmt.Errorf("bad configuration:\n%+v", cfg))
	}

	level, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	zcfg := zap.NewProductionConfig()
	zcfg.EncoderConfig.TimeKey = "timestamp"
	zcfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zcfg.Level = level

	log, err := zcfg.Build()
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	log.Info("metal-core version", zap.Any("version", v.V))

	log.Info("configuration",
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
		zap.String("gRPC-address", cfg.GrpcAddress),
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
			Log:             log,
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)
	app.SetEndpointHandler(endpoint.NewHandler)
	app.InitHMAC()
	app.SetEventHandler(event.NewHandler)

	err = app.initConsumer()
	if err != nil {
		log.Fatal("failed to init NSQ consumer", zap.Error(err))
	}

	s, err := app.APIClient().RegisterSwitch()
	if err != nil {
		log.Fatal("failed to register switch", zap.Error(err))
	}
	cert, err := os.ReadFile(cfg.GrpcCACertFile)
	if err != nil {
		log.Fatal("failed to read cert", zap.Error(err))
	}
	cacert, err := os.ReadFile(cfg.GrpcCACertFile)
	if err != nil {
		log.Fatal("failed to read cacert", zap.Error(err))
	}
	key, err := os.ReadFile(cfg.GrpcClientKeyFile)
	if err != nil {
		log.Fatal("failed to read key", zap.Error(err))
	}

	grpcClient, err := NewGrpcClient(log.Sugar(), cfg.GrpcAddress, cert, key, cacert)
	if err != nil {
		log.Fatal("failed to create grpc client", zap.Error(err))
	}
	eventServiceClient, err := grpcClient.NewEventClient()
	if err != nil {
		log.Fatal("failed to create grpc event service client", zap.Error(err))
	}
	app.SetEventServiceClient(eventServiceClient)

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
