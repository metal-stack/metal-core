package int

import (
	"bytes"
	"context"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var (
	apiServer *http.Server
	srv       core.Service
	logOutput bytes.Buffer
)

func runMetalCoreServer() {
	logOutput.Reset()
	log.SetOutput(&logOutput)

	os.Setenv("METAL_CORE_CONTROL_PLANE_IP", "localhost")
	os.Setenv("METAL_CORE_FACILITY_ID", "FRA")
	os.Setenv("METAL_CORE_PORT", "10000")
	os.Setenv("METAL_CORE_METAL_API_PORT", "10001")
	config := domain.Config{}
	if err := envconfig.Process("METAL_CORE", &config); err != nil {
		fmt.Println("Cannot fetch configuration")
		os.Exit(-1)
	}
	srv = core.NewService(&config)

	go func() {
		srv.RunServer()
	}()
	time.Sleep(200 * time.Millisecond)
}

func runMetalAPIMockServer(router *mux.Router) {
	if srv == nil {
		runMetalCoreServer()
	}
	apiServer = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", srv.GetConfig().APIPort),
		Handler: router,
	}
	go func() {
		if err := apiServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
}

func shutdown() {
	_shutdown(srv.GetServer())
	_shutdown(apiServer)
}

func _shutdown(server *http.Server) {
	if server != nil {
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}
}
