//go:build client
// +build client

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	clientv2 "github.com/metal-stack/api/go/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/metal-stack/metal-core/cmd/internal/core"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/v"
)

const phonedHomeInterval = time.Minute // lldpd sends messages every two seconds

func Run() {
	cfg := &Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		panic(fmt.Errorf("bad configuration:%w", err))
	}

	lvl := slog.LevelInfo
	switch strings.ToLower(cfg.LogLevel) {
	case "info":
		lvl = slog.LevelInfo
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl, AddSource: true}))

	log.Info("metal-core version", "version", v.V)
	log.Info("configuration", "cfg", cfg)

	driver, err := metalgo.NewDriver(
		fmt.Sprintf("%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath),
		"", cfg.HMACKey, metalgo.AuthType("Metal-Edit"),
	)
	if err != nil {
		log.Error("unable to create metal-api driver", "error", err)
		os.Exit(1)
	}

	client, err := newApiClient(cfg.ApiURL, cfg.ApiToken)
	if err != nil {
		log.Error("failed to create metal-apiserver client", "error", err)
		os.Exit(1)
	}

	cert, err := os.ReadFile(cfg.GrpcClientCertFile)
	if err != nil {
		log.Error("failed to read cert", "error", err)
		os.Exit(1)
	}
	cacert, err := os.ReadFile(cfg.GrpcCACertFile)
	if err != nil {
		log.Error("failed to read ca cert", "error", err)
		os.Exit(1)
	}
	key, err := os.ReadFile(cfg.GrpcClientKeyFile)
	if err != nil {
		log.Error("failed to read key", "error", err)
		os.Exit(1)
	}

	grpcClient, err := NewGrpcClient(log, cfg.GrpcAddress, cert, key, cacert)
	if err != nil {
		log.Error("failed to create grpc client", "error", err)
		os.Exit(1)
	}

	nos, err := switcher.NewNOS(log, cfg.FrrTplFile, cfg.InterfacesTplFile)
	if err != nil {
		log.Error("failed to create NOS instance", "error", err)
		os.Exit(1)
	}

	metrics := metrics.New()

	c := core.New(core.Config{
		Log:                   log,
		LogLevel:              cfg.LogLevel,
		CIDR:                  cfg.CIDR,
		LoopbackIP:            cfg.LoopbackIP,
		ASN:                   cfg.ASN,
		PartitionID:           cfg.PartitionID,
		RackID:                cfg.RackID,
		ReconfigureSwitch:     cfg.ReconfigureSwitch,
		ManagementGateway:     cfg.ManagementGateway,
		AdditionalBridgePorts: cfg.AdditionalBridgePorts,
		AdditionalBridgeVIDs:  cfg.AdditionalBridgeVIDs,
		SpineUplinks:          cfg.SpineUplinks,
		NOS:                   nos,
		Driver:                driver,
		Client:                client,
		EventServiceClient:    grpcClient.NewEventClient(),
		Metrics:               metrics,
		PXEVlanID:             cfg.PXEVlanID,
		BGPNeighborStateFile:  cfg.BGPNeighborStateFile,
	})

	err = c.RegisterSwitch()
	if err != nil {
		log.Error("failed to register switch", "error", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ConstantlyReconfigureSwitch(ctx, cfg.ReconfigureSwitchInterval)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ConstantlyPhoneHome(ctx, phonedHomeInterval)
	}()

	// Start metrics
	metricsAddr := fmt.Sprintf("%v:%d", cfg.MetricsServerBindAddress, cfg.MetricsServerPort)

	log.Info("starting metrics endpoint", "addr", metricsAddr)
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))
	metrics.Init()

	srv := &http.Server{
		Addr:              metricsAddr,
		Handler:           metricsServer,
		ReadHeaderTimeout: 3 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("unable to start metrics listener", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = srv.Shutdown(context.Background()); err != nil {
			log.Error("unable to shutdown metrics listener", "error", err)
		}
	}()

	wg.Wait()
}

func newApiClient(apiURL, token string) (clientv2.Client, error) {
	dialConfig := &clientv2.DialConfig{
		BaseURL:   apiURL,
		Token:     token,
		UserAgent: "metal-stack-cli",
		Log:       slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	return clientv2.New(dialConfig)
}
