//go:build client
// +build client

package cmd

import (
	"fmt"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"

	"github.com/kelseyhightower/envconfig"
	"github.com/metal-stack/metal-core/cmd/internal/core"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/v"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run() {
	cfg := &Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		panic(fmt.Errorf("bad configuration:%w", err))
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

	driver, err := metalgo.NewDriver(
		fmt.Sprintf("%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath),
		"", cfg.HMACKey, metalgo.AuthType("Metal-Edit"))

	if err != nil {
		log.Fatalw("unable to create metal-api driver", "error", err)
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

	var nos core.NOS
	if _, err := os.Stat(sonic.SonicVersionFile); err == nil {
		nos, err = sonic.NewSonic(log)
		if err != nil {
			log.Fatalw("failed to initialize SONiC configurator", "error", err)
		}
	} else {
		nos = cumulus.NewCumulus(log, cfg.FrrTplFile, cfg.InterfacesTplFile)
	}

	c := core.New(core.Config{
		Log:                       log,
		LogLevel:                  cfg.LogLevel,
		CIDR:                      cfg.CIDR,
		LoopbackIP:                cfg.LoopbackIP,
		ASN:                       cfg.ASN,
		PartitionID:               cfg.PartitionID,
		RackID:                    cfg.RackID,
		ReconfigureSwitch:         cfg.ReconfigureSwitch,
		ReconfigureSwitchInterval: cfg.ReconfigureSwitchInterval,
		ManagementGateway:         cfg.ManagementGateway,
		AdditionalBridgePorts:     cfg.AdditionalBridgePorts,
		AdditionalBridgeVIDs:      cfg.AdditionalBridgeVIDs,
		SpineUplinks:              cfg.SpineUplinks,
		DHCPServers:               cfg.DHCPServers,
		NOS:                       nos,
		Driver:                    driver,
		EventServiceClient:        grpcClient.NewEventClient(),
	})
	err = c.RegisterSwitch()
	if err != nil {
		log.Fatalw("failed to register switch", "error", err)
	}

	go c.ReconfigureSwitch()
	c.ConstantlyPhoneHome()

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		_ = os.Setenv("DEBUG", "1")
	}

	// Start metrics
	metricsAddr := fmt.Sprintf("%v:%d", cfg.MetricsServerBindAddress, cfg.MetricsServerPort)

	log.Infow("starting metrics endpoint", "addr", metricsAddr)
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))

	log.Fatal(http.ListenAndServe(metricsAddr, metricsServer))
}
