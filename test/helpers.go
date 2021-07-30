package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/metal-stack/go-hal/pkg/api"

	"github.com/emicklei/go-restful/v3"
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

func (h *noopEventHandler) PowerBootBiosMachine(machineID string) {}

func (h *noopEventHandler) PowerBootDiskMachine(machineID string) {}

func (h *noopEventHandler) PowerBootPxeMachine(machineID string) {}

func (h *noopEventHandler) ReinstallMachine(machineID string) {}

func (h *noopEventHandler) PowerOnChassisIdentifyLED(machineID, description string) {}

func (h *noopEventHandler) PowerOffChassisIdentifyLED(machineID, description string) {}

func (h *noopEventHandler) UpdateBios(machineID, revision, description string, s3Cfg *api.S3Config) {}

func (h *noopEventHandler) UpdateBmc(machineID, revision, description string, s3Cfg *api.S3Config) {}

func (h *noopEventHandler) TriggerSwitchReconfigure(switchName, eventType string) {}

func (h *noopEventHandler) ReconfigureSwitch() {}

func mockAPIEndpoint(apiClient func(ctx *domain.AppContext) domain.APIClient) domain.EndpointHandler {
	_ = os.Setenv(zapup.KeyLogLevel, "info")
	_ = os.Setenv(zapup.KeyOutput, logFilename)
	_ = os.Setenv("METAL_CORE_CIDR", "10.0.0.11/24")
	_ = os.Setenv("METAL_CORE_PARTITION_ID", "FRA")
	_ = os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
	_ = os.Setenv("METAL_CORE_HMAC_KEY", "blubber")
	_ = os.Setenv("METAL_CORE_GRPC_ADDRESS", "10.0.0.11:50051")

	cfg = &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		fmt.Println(err)
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

func doGet(path string, response interface{}) (int, error) {
	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, path, nil)
	r := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(r, req)
	if err := json.Unmarshal(r.Body.Bytes(), response); err != nil {
		return 0, err
	}
	return r.Result().StatusCode, nil //nolint
}

func doPost(path string, payload interface{}) int {
	bodyJSON, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, path, bytes.NewBuffer(bodyJSON))
	req.Header.Add("Content-Type", restful.MIME_JSON)
	r := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(r, req)
	return r.Result().StatusCode //nolint
}

func getLogs() (string, error) {
	logs, err := os.ReadFile(logFilename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(logs)), nil
}

func truncateLogFile() error {
	logFile, err := os.OpenFile(logFilename, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	defer func() {
		_ = logFile.Close()
	}()

	_ = logFile.Truncate(0)
	_, _ = logFile.Seek(0, 0)
	return nil
}

func deleteLogFile() {
	_ = os.Remove(logFilename)
}
