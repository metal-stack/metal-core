package metalcore

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/metal-stack/metal-core/internal/api"
	"github.com/metal-stack/metal-core/internal/bmc"
	"github.com/metal-stack/metal-core/internal/core"
	"github.com/metal-stack/metal-core/pkg/domain"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/v"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Server struct {
	*domain.AppContext
}

func Run() {
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

	l, err := zcfg.Build()
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	log := l.Sugar()
	log.Infow("metal-core version", "version", v.V)
	log.Infow("configuration", "cfg", cfg)

	driver, _, err := metalgo.NewDriver(
		fmt.Sprintf("%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath),
		"", cfg.HMACKey, metalgo.AuthType("Metal-Edit"))

	if err != nil {
		log.Fatalw("unable to create metal-api driver", "error", err)
	}

	app := &Server{
		AppContext: &domain.AppContext{
			Driver: driver,
			Config: cfg,
			Log:    l,
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)

	err = app.APIClient().RegisterSwitch()
	if err != nil {
		log.Fatalw("failed to register switch", "error", err)
	}
	cert, err := os.ReadFile(cfg.GrpcClientCertFile)
	if err != nil {
		log.Fatalw("failed to read cert", "error", err)
	}
	cacert, err := os.ReadFile(cfg.GrpcCACertFile)
	if err != nil {
		log.Fatalw("failed to read ca cert", "error", err)
	}
	key, err := os.ReadFile(cfg.GrpcClientKeyFile)
	if err != nil {
		log.Fatalw("failed to read key", "error", err)
	}

	grpcClient, err := NewGrpcClient(log, cfg.GrpcAddress, cert, key, cacert)
	if err != nil {
		log.Fatalw("failed to create grpc client", "error", err)
	}
	app.SetEventServiceClient(grpcClient.NewEventClient())

	go app.APIClient().ReconfigureSwitch()
	app.APIClient().ConstantlyPhoneHome()

	b := bmc.New(bmc.Config{
		Log:              l,
		MQAddress:        cfg.MQAddress,
		MQCACertFile:     cfg.MQCACertFile,
		MQClientCertFile: cfg.MQClientCertFile,
		MQLogLevel:       cfg.MQLogLevel,
		MachineTopic:     cfg.MachineTopic,
		MachineTopicTTL:  cfg.MachineTopicTTL,
	})
	err = b.InitConsumer()
	if err != nil {
		log.Fatalw("unable to create bmcservice", "error", err)
	}

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	app.Server().Run()
}
