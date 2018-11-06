package int

import (
	"context"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"github.com/kelseyhightower/envconfig"
	"io/ioutil"
	"log"
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
	apiServer *http.Server
	srv       core.Service
)

func runMetalCoreServer() {
	os.Setenv(zapup.KeyOutput, logFilename)
	if srv != nil {
		return
	}
	os.Setenv("METAL_CORE_IP", "127.0.0.1")
	os.Setenv("METAL_CORE_SITE_ID", "FRA")
	os.Setenv("METAL_CORE_RACK_ID", "Vagrant Rack 1")
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

func mockMetalAPIServer(endpoints ...endpoint) {
	handler := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	for _, e := range endpoints {
		ws.Route(ws.Method(e.method).Path(e.path).To(e.handler))
	}
	handler.Add(ws)

	apiServer = &http.Server{
		Addr:    fmt.Sprintf("%v:%d", srv.GetConfig().ApiIP, srv.GetConfig().ApiPort),
		Handler: handler,
	}
	go func() {
		if err := apiServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
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
