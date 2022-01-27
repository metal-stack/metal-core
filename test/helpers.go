package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/metal-stack/go-hal/pkg/api"
	"go.uber.org/zap"

	"github.com/emicklei/go-restful/v3"
	"github.com/kelseyhightower/envconfig"
	ep "github.com/metal-stack/metal-core/internal/endpoint"
	"github.com/metal-stack/metal-core/pkg/domain"
)

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

func (h *noopEventHandler) PowerCycleMachine(machineID string) {}

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

	log, _ := zap.NewProduction()

	appContext = &domain.AppContext{
		Config:     cfg,
		BootConfig: bootCfg,
		Log:        log,
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
