package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	ep "git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"github.com/kelseyhightower/envconfig"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

const logFilename = "output.log"

var (
	cfg        *domain.Config
	appContext *domain.AppContext
)

func mockApiEndpoint(apiHandler func(ctx *domain.AppContext) domain.APIClient) domain.Endpoint {
	os.Setenv(zapup.KeyOutput, logFilename)
	os.Setenv("METAL_CORE_IP", "test-host")
	os.Setenv("METAL_CORE_SITE_ID", "FRA")
	os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
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
		Config:           cfg,
		BootConfig:       bootCfg,
		ApiClientHandler: apiHandler,
		EndpointHandler:  ep.Handler,
	}
	return appContext.Endpoint()
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
	defer logFile.Close()

	logFile.Truncate(0)
	logFile.Seek(0, 0)
}

func deleteLogFile() {
	os.Remove(logFilename)
}
