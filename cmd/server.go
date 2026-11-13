//go:build client

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	httppprof "net/http/pprof"
	"net/netip"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	clientv2 "github.com/metal-stack/api/go/client"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/samber/lo"
	"go.yaml.in/yaml/v4"

	"github.com/metal-stack/metal-core/cmd/internal/core"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/v"
)

const phonedHomeInterval = time.Minute

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

	for _, route := range cfg.AdditionalMgmtRoutes {
		if _, err := netip.ParsePrefix(route); err != nil {
			log.Error("invalid cidr in additional mgmt routes", "cidr", route, "error", err)
			os.Exit(1)
		}
	}

	client, err := newApiClient(log, cfg.ApiURL, cfg.ApiserverTokenFile)
	if err != nil {
		log.Error("failed to create metal-apiserver client", "error", err)
		os.Exit(1)
	}

	client.Ping(context.Background(), &clientv2.PingConfig{
		ComponentType: apiv2.ComponentType_COMPONENT_TYPE_METAL_CORE,
		Version: apiv2.Version{
			Version:   v.Version,
			Revision:  v.Revision,
			GitSha1:   v.GitSHA1,
			BuildDate: v.BuildDate,
		},
	})

	nos, err := switcher.NewNOS(log, cfg.FrrTplFile, cfg.InterfacesTplFile, sonic.InterfaceNamingSchema(cfg.InterfaceNamingSchema))
	if err != nil {
		log.Error("failed to create NOS instance", "error", err)
		os.Exit(1)
	}

	frrConf, err := getStaticBGPConfig(cfg.StatigBGPConfigFile)
	if err != nil {
		log.Error("failed to get static bgp configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nics, err := nos.GetNics(ctx)
	if err != nil {
		log.Error("failed to get switch nics", "error", err)
		os.Exit(1)
	}

	err = validateStaticBGPConfig(frrConf, nics)
	if err != nil {
		log.Error("validate static frr config failed", "error", err)
		os.Exit(1)
	}

	metrics := metrics.New()
	c := core.New(core.Config{
		Log:                  log,
		LogLevel:             cfg.LogLevel,
		CIDR:                 cfg.CIDR,
		LoopbackIP:           cfg.LoopbackIP,
		ASN:                  cfg.ASN,
		PartitionID:          cfg.PartitionID,
		RackID:               cfg.RackID,
		RoomID:               cfg.RoomID,
		ReconfigureSwitch:    cfg.ReconfigureSwitch,
		ManagementGateway:    cfg.ManagementGateway,
		AdditionalMgmtRoutes: cfg.AdditionalMgmtRoutes,
		SpineUplinks:         cfg.SpineUplinks,
		SetSrcLoopback:       cfg.SetSrcLoopback,
		NOS:                  nos,
		Client:               client,
		Metrics:              metrics,
		PXEVlanID:            cfg.PXEVlanID,
		BGPNeighborStateFile: cfg.BGPNeighborStateFile,
		StaticFRRConfig:      frrConf,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = c.RegisterSwitch(ctx, cfg.Timeout)
	if err != nil {
		log.Error("failed to register switch", "error", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		c.ConstantlyReconfigureSwitch(ctx, cfg.ReconfigureSwitchInterval, cfg.Timeout)
	})
	wg.Go(func() {
		c.ConstantlyPhoneHome(ctx, phonedHomeInterval)
	})

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

	wg.Go(func() {
		if err = srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("unable to start metrics listener", "error", err)
			os.Exit(1)
		}
	})

	<-ctx.Done()

	wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = srv.Shutdown(ctx); err != nil {
			log.Error("unable to shutdown metrics listener", "error", err)
		}
	})

	wg.Wait()
}

func getStaticBGPConfig(file string) (*types.FRRConfig, error) {
	config := &types.FRRConfig{}

	fileBytes, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read static VRFs file: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(fileBytes))
	dec.KnownFields(true)
	err = dec.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal static VRFs file: %w", err)
	}

	return config, nil
}

func validateStaticBGPConfig(conf *types.FRRConfig, nics []*apiv2.SwitchNic) error {
	for _, vrf := range conf.VRFs {
		if _, err := types.ParseVNI(vrf); err != nil {
			return err
		}
	}

	if err := validateUnnumberedInterfaces(conf, nics); err != nil {
		return err
	}

	if err := validateBGPNeighbors(conf); err != nil {
		return err
	}

	if err := validateAddressFamiliesCommands(conf); err != nil {
		return err
	}

	for _, bgp := range conf.BGPs {
		if err := validateAddressFamiliesNeighbors(bgp); err != nil {
			return err
		}
	}

	if err := validateRouteMaps(conf); err != nil {
		return err
	}

	if err := validatePrefixLists(conf.PrefixLists); err != nil {
		return err
	}

	return nil
}

func validateUnnumberedInterfaces(conf *types.FRRConfig, nics []*apiv2.SwitchNic) error {
	for _, intf := range conf.UnnumberedInterfaces {
		if !slices.ContainsFunc(nics, func(n *apiv2.SwitchNic) bool {
			return n.Name == intf.Name
		}) {
			return fmt.Errorf("unnumbered interface %q is missing from switch nics", intf.Name)
		}

		if intf.Vrf == nil {
			continue
		}

		if !slices.ContainsFunc(conf.VRFs, func(v string) bool {
			return pointer.SafeDeref(intf.Vrf) == v
		}) {
			return fmt.Errorf("unnumbered interface %q is bound to vrf %q which is missing from vrf config", intf.Name, pointer.SafeDeref(intf.Vrf))
		}
	}
	return nil
}

func validateBGPNeighbors(conf *types.FRRConfig) error {
	for _, bgp := range conf.BGPs {
		for _, neigh := range bgp.Neighbors {
			trimmed, found := strings.CutSuffix(neigh.Name, " interface")
			if found {
				intf, ok := lo.Find(conf.UnnumberedInterfaces, func(i *types.Interface) bool {
					return i.Name == trimmed
				})
				if !ok {
					return fmt.Errorf("unnumbered neighbor %q is missing from unnumbered interfaces", trimmed)
				}

				if pointer.SafeDeref(intf.Vrf) != pointer.SafeDeref(bgp.Vrf) {
					return fmt.Errorf("unnumbered neighbor %q is bound to vrf %q but is added as neighbor in vrf %q", trimmed, pointer.SafeDeref(intf.Vrf), pointer.SafeDeref(bgp.Vrf))
				}
			}

			if !found {
				_, err := netip.ParseAddr(trimmed)
				if err != nil {
					return fmt.Errorf(`neighbor %q is not a valid ip address; if you are trying to add an unnumbered neighbor add the suffix "interface": "%s interface"`, trimmed, trimmed)
				}

			}

			if neigh.PeerGroup != nil {
				_, found := lo.Find(bgp.PeerGroups, func(p types.PeerGroup) bool {
					return p.Name == pointer.SafeDeref(neigh.PeerGroup)
				})
				if !found {
					return fmt.Errorf("peer-group %q for neighbor %q does not exist", pointer.SafeDeref(neigh.PeerGroup), neigh.Name)
				}
			}
		}
	}

	return nil
}

func validateAddressFamiliesCommands(conf *types.FRRConfig) error {
	for _, bgp := range conf.BGPs {
		if af := bgp.AddressFamilies.IPv4Unicast; af != nil {
			if err := validateCommandRouteMaps(af.GlobalCommands, conf.RouteMaps); err != nil {
				return fmt.Errorf(`global commands for address-family "ipv4 unicast" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
			}

			for _, neigh := range af.Neighbors {
				if err := validateCommandRouteMaps(neigh.Commands, conf.RouteMaps); err != nil {
					return fmt.Errorf(`neighbor commands for neighbor %q for address-family "ipv4 unicast" in vrf %q failed: %w`, neigh.Name, pointer.SafeDeref(bgp.Vrf), err)
				}
			}
		}
		if af := bgp.AddressFamilies.IPv6Unicast; af != nil {
			if err := validateCommandRouteMaps(af.GlobalCommands, conf.RouteMaps); err != nil {
				return fmt.Errorf(`global commands for address-family "ipv6 unicast" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
			}

			for _, neigh := range af.Neighbors {
				if err := validateCommandRouteMaps(neigh.Commands, conf.RouteMaps); err != nil {
					return fmt.Errorf(`neighbor commands for neighbor %q for address-family "ipv6 unicast" in vrf %q failed: %w`, neigh.Name, pointer.SafeDeref(bgp.Vrf), err)
				}
			}
		}
		if af := bgp.AddressFamilies.L2VPNEVPN; af != nil {
			if err := validateCommandRouteMaps(af.GlobalCommands, conf.RouteMaps); err != nil {
				return fmt.Errorf(`global commands for address-family "l2vpn evpn" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
			}

			for _, neigh := range af.Neighbors {
				if err := validateCommandRouteMaps(neigh.Commands, conf.RouteMaps); err != nil {
					return fmt.Errorf(`neighbor commands for neighbor %q for address-family "l2vpn evpn" in vrf %q failed: %w`, neigh.Name, pointer.SafeDeref(bgp.Vrf), err)
				}
			}
		}
	}
	return nil
}

func validateCommandRouteMaps(cmds []string, rms []*types.RouteMap) error {
	for _, cmd := range cmds {
		split := strings.Split(cmd, "route-map")
		if len(split) == 1 && split[0] == cmd {
			continue
		}
		rm := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(split[len(split)-1], "in"), "out"))
		if !slices.ContainsFunc(rms, func(r *types.RouteMap) bool {
			return r.Name == rm
		}) {
			return fmt.Errorf(`command %q references route-map %q which is not configured`, cmd, rm)
		}
	}
	return nil
}

func validateAddressFamiliesNeighbors(bgp *types.BGP) error {
	if af := bgp.AddressFamilies.IPv4Unicast; af != nil {
		if err := validateAddressFamilyNeighbors(af.Neighbors, bgp.PeerGroups, bgp.Neighbors); err != nil {
			return fmt.Errorf(`neighbor validation for address-family "ipv4 unicast" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
		}
	}
	if af := bgp.AddressFamilies.IPv6Unicast; af != nil {
		if err := validateAddressFamilyNeighbors(af.Neighbors, bgp.PeerGroups, bgp.Neighbors); err != nil {
			return fmt.Errorf(`neighbor validation for address-family "ipv6 unicast" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
		}
	}
	if af := bgp.AddressFamilies.L2VPNEVPN; af != nil {
		if err := validateAddressFamilyNeighbors(af.Neighbors, bgp.PeerGroups, bgp.Neighbors); err != nil {
			return fmt.Errorf(`neighbor validation for address-family "l2vpn evpn" in vrf %q failed: %w`, pointer.SafeDeref(bgp.Vrf), err)
		}
	}
	return nil
}

func validateAddressFamilyNeighbors(afNeighs []*types.AddressFamilyNeighbor, peerGroups []types.PeerGroup, neighs []types.BGPNeighbor) error {
	for _, afNeigh := range afNeighs {
		if slices.ContainsFunc(neighs, func(n types.BGPNeighbor) bool {
			return strings.TrimSuffix(n.Name, " interface") == afNeigh.Name
		}) {
			continue
		}

		if !slices.ContainsFunc(peerGroups, func(pg types.PeerGroup) bool {
			return pg.Name == afNeigh.Name
		}) {
			return fmt.Errorf("neighbor %q is neither a peer-group nor a bgp neighbor", afNeigh.Name)
		}
	}
	return nil
}

func validateRouteMaps(conf *types.FRRConfig) error {
	for i, rm1 := range conf.RouteMaps {
		for j, rm2 := range conf.RouteMaps {
			if i == j {
				continue
			}
			if rm1.Name == rm2.Name && rm1.Order == rm2.Order {
				return fmt.Errorf("route-map %q is defined multiple times with the same order %d", rm1.Name, rm1.Order)
			}
		}
	}

	for _, rm := range conf.RouteMaps {
		if rm.Policy != types.PolicyDeny && rm.Policy != types.PolicyPermit {
			return fmt.Errorf("route-map %q has policy %q, must be %q or %q", rm.Name, rm.Policy, types.PolicyDeny, types.PolicyPermit)
		}

		if rm.Order <= 0 {
			return fmt.Errorf("route-map %q has order %d, choose an order > 0", rm.Name, rm.Order)
		}

		for _, entry := range rm.Entries {
			split := strings.Split(entry, "prefix-list")
			if len(split) == 0 {
				continue
			}

			pf := strings.TrimSpace(split[len(split)-1])
			if !slices.ContainsFunc(conf.PrefixLists, func(p *types.IPPrefixList) bool {
				return p.Name == pf
			}) {
				return fmt.Errorf("route-map %q references prefix-list %q which is not defined", rm.Name, pf)
			}
		}
	}

	return nil
}

func validatePrefixLists(pfs []*types.IPPrefixList) error {
	for _, pf := range pfs {
		split := strings.Split(pf.Spec, string(types.PolicyDeny))
		if len(split) == 1 {
			split = strings.Split(pf.Spec, string(types.PolicyPermit))
		}
		if len(split) < 1 {
			continue
		}

		cidrAndSize := split[len(split)-1]

		split = strings.Split(cidrAndSize, "le")
		if len(split) == 1 {
			split = strings.Split(cidrAndSize, "ge")
		}
		if len(split) < 1 {
			continue
		}

		cidr := strings.TrimSpace(split[0])
		var sizeStr string
		if len(split) == 2 {
			sizeStr = strings.TrimSpace(split[1])
		}

		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return fmt.Errorf("failed to parse prefix %q of prefix-list %q: %w", cidr, pf.Name, err)
		}

		if pf.AddressFamily == types.AddressFamilyIPv6 {
			if prefix.Addr().Is4() {
				return fmt.Errorf("prefix-list %q is an ipv6 prefix-list but references an ipv4 cidr %q", pf.Name, cidr)
			}

			if sizeStr != "" {
				size, err := strconv.Atoi(sizeStr)
				if err != nil {
					return fmt.Errorf("failed to parse mask size %q of prefix-list %q: %w", sizeStr, pf.Name, err)
				}
				if size < 0 || size > 128 {
					return fmt.Errorf("prefix-list %q uses a mask size of %d, where it must be 0 <= size <= 128", pf.Name, size)
				}
			}
		}
		if pf.AddressFamily == types.AddressFamilyIPv4 {
			if prefix.Addr().Is6() {
				return fmt.Errorf("prefix-list %q is an ipv4 prefix-list but references an ipv6 cidr %q", pf.Name, cidr)
			}

			if sizeStr != "" {
				size, err := strconv.Atoi(sizeStr)
				if err != nil {
					return fmt.Errorf("failed to parse mask size %q of prefix-list %q: %w", sizeStr, pf.Name, err)
				}
				if size < 0 || size > 32 {
					return fmt.Errorf("prefix-list %q uses a mask size of %d, where it must be 0 <= size <= 32", pf.Name, size)
				}
			}
		}
	}
	return nil
}

func newApiClient(logger *slog.Logger, apiURL, tokenfilePath string) (clientv2.Client, error) {
	token, err := os.ReadFile(tokenfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file %s: %w", tokenfilePath, err)
	}
	tokenPersister, err := clientv2.NewFilesystemTokenPersister(tokenfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem token persister: %w", err)
	}

	dialConfig := &clientv2.DialConfig{
		BaseURL:   apiURL,
		Token:     string(token),
		UserAgent: "metal-core",
		Log:       logger.WithGroup("client"),
		TokenRenewal: &clientv2.TokenRenewal{
			PersistTokenFn: tokenPersister,
		},
	}

	return clientv2.New(dialConfig)
}
