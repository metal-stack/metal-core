package metalcore

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/metal-stack/metal-core/internal/api"
	"github.com/metal-stack/metal-core/internal/core"
	"github.com/metal-stack/metal-core/internal/event"
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

	log, err := zcfg.Build()
	if err != nil {
		panic(fmt.Errorf("can't initialize zap logger: %w", err))
	}

	log.Info("metal-core version", zap.Any("version", v.V))

	log.Sugar().Infow("configuration", "cfg", cfg)

	driver, _, err := metalgo.NewDriver(
		fmt.Sprintf("%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath),
		"", cfg.HMACKey, metalgo.AuthType("Metal-Edit"))

	if err != nil {
		log.Sugar().Fatalw("unable to create metal-api driver", "error", err)
	}

	app := &Server{
		AppContext: &domain.AppContext{
			Driver: driver,
			Config: cfg,
			Log:    log,
		},
	}
	app.SetAPIClient(api.NewClient)
	app.SetServer(core.NewServer)
	app.SetEventHandler(event.NewHandler)

	err = app.initConsumer()
	if err != nil {
		log.Fatal("failed to init NSQ consumer", zap.Error(err))
	}

	err = app.APIClient().RegisterSwitch()
	if err != nil {
		log.Fatal("failed to register switch", zap.Error(err))
	}
	cert, err := os.ReadFile(cfg.GrpcClientCertFile)
	if err != nil {
		log.Fatal("failed to read cert", zap.Error(err))
	}
	cacert, err := os.ReadFile(cfg.GrpcCACertFile)
	if err != nil {
		log.Fatal("failed to read ca cert", zap.Error(err))
	}
	key, err := os.ReadFile(cfg.GrpcClientKeyFile)
	if err != nil {
		log.Fatal("failed to read key", zap.Error(err))
	}

	grpcClient, err := NewGrpcClient(log.Sugar(), cfg.GrpcAddress, cert, key, cacert)
	if err != nil {
		log.Fatal("failed to create grpc client", zap.Error(err))
	}
	app.SetEventServiceClient(grpcClient.NewEventClient())

	app.initSwitchReconfiguration()
	app.APIClient().ConstantlyPhoneHome()

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	app.Server().Run()
}
