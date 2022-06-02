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

type App struct {
	ctx *domain.AppContext
}

func (a *App) Run() {
	a.startConstantlySwitchReconfiguration()
	a.startConstantlyPhoneHome()
	a.runServer()
}

func (a *App) startConstantlySwitchReconfiguration() {
	go a.ctx.EventHandler().ReconfigureSwitch()
}

func (a *App) startConstantlyPhoneHome() {
	a.ctx.APIClient().ConstantlyPhoneHome()
}

func (a *App) runServer() {
	a.ctx.Server().Run()
}

func Create() *App {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		panic(fmt.Errorf("bad configuration:\n%+v", cfg))
	}

	level := zap.InfoLevel
	err := level.UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	zcfg := zap.NewProductionConfig()
	zcfg.EncoderConfig.TimeKey = "timestamp"
	zcfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zcfg.Level = zap.NewAtomicLevelAt(level)

	log, err := zcfg.Build()
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	log.Info("metal-core version", zap.Any("version", v.V))

	devMode := strings.Contains(cfg.PartitionID, "vagrant")

	logConfiguration(log, devMode, cfg)

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), cfg.ApiBasePath, []string{cfg.ApiProtocol})

	ctx := &domain.AppContext{
		Config:          cfg,
		MachineClient:   machine.New(transport, strfmt.Default),
		PartitionClient: partition.New(transport, strfmt.Default),
		SwitchClient:    sw.New(transport, strfmt.Default),
		DevMode:         devMode,
		Log:             log,
	}
	ctx.SetAPIClient(api.NewClient)
	ctx.SetServer(core.NewServer)
	ctx.SetEndpointHandler(endpoint.NewHandler)
	ctx.InitHMAC()
	ctx.SetEventHandler(event.NewHandler)

	mqClient := newMQClient(cfg, log)

	err = mqClient.initConsumer(ctx.EventHandler())
	if err != nil {
		log.Fatal("failed to init NSQ consumer",
			zap.Error(err),
		)
		os.Exit(1)
	}

	s, err := ctx.APIClient().RegisterSwitch()
	if err != nil {
		log.Fatal("failed to register switch",
			zap.Error(err),
		)
		os.Exit(1)
	}

	ctx.BootConfig = &domain.BootConfig{
		MetalHammerImageURL:    s.Partition.Bootconfig.Imageurl,
		MetalHammerKernelURL:   s.Partition.Bootconfig.Kernelurl,
		MetalHammerCommandLine: s.Partition.Bootconfig.Commandline,
	}

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	return &App{ctx: ctx}
}

func logConfiguration(log *zap.Logger, devMode bool, cfg *domain.Config) {
	log.Info("configuration",
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
		zap.String("gRPC-address", cfg.GrpcAddress),
		zap.String("gRPC-CACertFile", cfg.GrpcCACertFile),
		zap.String("gRPC-clientCertFile", cfg.GrpcClientCertFile),
		zap.String("gRPC-clientKeyFile", cfg.GrpcClientKeyFile),
	)
}
