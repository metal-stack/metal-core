package test

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	ep "git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/server"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kelseyhightower/envconfig"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const logFilename = "output.log"

var (
	cfg        *domain.Config
	appContext *domain.AppContext
)

func runMetalCoreServer(apiHandler func(ctx *domain.AppContext) domain.APIClient) {
	if cfg != nil {
		appContext.ApiClientHandler = apiHandler
		return
	}
	os.Setenv(zapup.KeyOutput, logFilename)
	os.Setenv("METAL_CORE_IP", "127.0.0.1")
	os.Setenv("METAL_CORE_SITE_ID", "FRA")
	os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
	os.Setenv("METAL_CORE_PORT", "10000")
	cfg = &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		fmt.Println("Cannot fetch configuration")
		os.Exit(-1)
	}

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	appContext = &domain.AppContext{
		Config:              cfg,
		ApiClientHandler:    apiHandler,
		ServerHandler:       server.Handler,
		EndpointHandler:     ep.Handler,
		EventHandlerHandler: event.Handler,
		DeviceClient:        device.New(transport, strfmt.Default),
	}

	go func() {
		appContext.Server().Run()
	}()
	time.Sleep(100 * time.Millisecond)
}

func getLogs() string {
	logs, err := ioutil.ReadFile(logFilename)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(logs))
}

func truncateLogFile() {
	logFile, err := os.OpenFile(logFilename, os.O_RDWR, 0666)
	if err != nil {
		return
	}
	defer logFile.Close()

	logFile.Truncate(0)
	logFile.Seek(0, 0)
}

func deleteLogFile() {
	os.Remove(logFilename)
}
