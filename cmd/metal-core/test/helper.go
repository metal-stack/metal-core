package test

import (
	"context"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/api"
	ep "git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/server"
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kelseyhightower/envconfig"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type endpoint struct {
	path    string
	handler func(request *restful.Request, response *restful.Response)
	method  string
}

const logFilename = "output.log"

var (
	apiServer  *http.Server
	cfg        *domain.Config
	appContext *domain.AppContext
)

func runMetalCoreServer() {
	if cfg != nil {
		return
	}
	os.Setenv(zapup.KeyOutput, logFilename)
	os.Setenv("METAL_CORE_IP", "127.0.0.1")
	os.Setenv("METAL_CORE_SITE_ID", "FRA")
	os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
	os.Setenv("METAL_CORE_PORT", "10000")
	os.Setenv("METAL_CORE_METAL_API_PORT", "10001")
	cfg = &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		fmt.Println("Cannot fetch configuration")
		os.Exit(-1)
	}

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	appContext = &domain.AppContext{
		Config:           cfg,
		ApiClientHandler: api.Handler,
		ServerHandler:    server.Handler,
		EndpointHandler:  ep.Handler,
		EventHandler:     event.Handler,
		DeviceClient:     device.New(transport, strfmt.Default),
		IpmiConnection: &domain.IpmiConnection{
			// Requires gateway of the control plane for running in Metal Lab... this is just a quick workaround for the poc
			Hostname:  cfg.IP[:strings.LastIndex(cfg.IP, ".")] + ".1",
			Interface: "lanplus",
			Port:      6230,
			Username:  "vagrant",
			Password:  "vagrant",
		},
	}

	go func() {
		appContext.Server().Run()
	}()
	time.Sleep(100 * time.Millisecond)
}

func getLogs() string {
	if logs, err := ioutil.ReadFile(logFilename); err != nil {
		panic(err)
	} else {
		os.Remove(logFilename)
		os.Unsetenv(zapup.KeyOutput)
		return strings.TrimSpace(string(logs))
	}
}

func mockMetalAPIServer(ee ...endpoint) {
	handler := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	for _, e := range ee {
		ws.Route(ws.Method(e.method).Path(e.path).To(e.handler))
	}
	handler.Add(ws)

	apiServer = &http.Server{
		Addr:    fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort),
		Handler: handler,
	}
	go func() {
		if err := apiServer.ListenAndServe(); err != http.ErrServerClosed {
			zapup.MustRootLogger().Fatal(err.Error())
		}
	}()
	time.Sleep(100 * time.Millisecond)
}

func shutdown() {
	if apiServer != nil {
		if err := apiServer.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}
}
