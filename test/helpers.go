package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/kelseyhightower/envconfig"
	ep "github.com/metal-stack/metal-core/internal/endpoint"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
)

const logFilename = "output.log"

var (
	cfg        *domain.Config
	appContext *domain.AppContext
)

type noopEventHandler struct {
	*domain.AppContext
}

func newNoopEventHandler(ctx *domain.AppContext) domain.EventHandler {
	return &noopEventHandler{ctx}
}

func (h *noopEventHandler) FreeMachine(machineID string) {}

func (h *noopEventHandler) PowerOnMachine(machineID string) {}

func (h *noopEventHandler) PowerOffMachine(machineID string) {}

func (h *noopEventHandler) PowerResetMachine(machineID string) {}

func (h *noopEventHandler) BootBiosMachine(machineID string) {}

func (h *noopEventHandler) AbortReinstallMachine(machineID string) {}

func (h *noopEventHandler) PowerOnChassisIdentifyLED(machineID, description string) {}

func (h *noopEventHandler) PowerOffChassisIdentifyLED(machineID, description string) {}

func (h *noopEventHandler) ReconfigureSwitch(switchID string) error {
	return nil
}

func mockAPIEndpoint(apiClient func(ctx *domain.AppContext) domain.APIClient) domain.EndpointHandler {
	_ = os.Setenv(zapup.KeyLogLevel, "info")
	_ = os.Setenv(zapup.KeyOutput, logFilename)
	_ = os.Setenv("METAL_CORE_CIDR", "10.0.0.11/24")
	_ = os.Setenv("METAL_CORE_PARTITION_ID", "FRA")
	_ = os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
	_ = os.Setenv("METAL_CORE_HMAC_KEY", "blubber")

	cfg = &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		fmt.Println("Cannot fetch configuration")
		os.Exit(-1)
	}
	bootCfg := &domain.BootConfig{
		MetalHammerImageURL:    "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-initrd.img.lz4",
		MetalHammerKernelURL:   "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-kernel",
		MetalHammerCommandLine: "",
	}

	appContext = &domain.AppContext{
		Config:     cfg,
		BootConfig: bootCfg,
	}
	appContext.SetAPIClient(apiClient)
	appContext.SetEndpointHandler(ep.NewHandler)
	appContext.SetEventHandler(newNoopEventHandler)

	return appContext.EndpointHandler()
}

func doGet(path string, response interface{}) int {
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(rr, req)
	if err := json.Unmarshal(rr.Body.Bytes(), response); err != nil {
		panic(err)
	}
	return rr.Result().StatusCode
}

func doPost(path string, payload interface{}) int {
	bodyJSON, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(bodyJSON))
	req.Header.Add("Content-Type", restful.MIME_JSON)
	rr := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(rr, req)
	return rr.Result().StatusCode
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

	defer func() {
		_ = logFile.Close()
	}()

	_ = logFile.Truncate(0)
	_, _ = logFile.Seek(0, 0)
}

func deleteLogFile() {
	_ = os.Remove(logFilename)
}
