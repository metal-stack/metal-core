package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/tcpproxy"
	"github.com/magefile/mage/sh"
	"github.com/magiconair/properties"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type Env struct {
	MetalCorePort int `properties:"METAL_CORE_PORT"`
	MetalAPIPort  int `properties:"METAL_API_PORT"`
}

var (
	env         Env
	proxy       tcpproxy.Proxy
	coreTraffic []string
	apiTraffic  []string
)

func TestWorkflow(t *testing.T) {
	if os.Getegid() != 0 {
		if err := runAsRoot(); err != nil {
			panic(err)
		} else {
			os.Exit(0)
		}
	}
	defer tearDown()

	// GIVEN
	spawnMetalCoreRethinkdbAndMetalAPI()
	sniffTcpPackets()

	// WHEN
	start("metal-hammer")

	// THEN

	//fmt.Println("=== Core Traffic ======================================================")
	//for i, c := range coreTraffic {
	//	fmt.Printf("---- %d ----------------------------\n", i)
	//	fmt.Println(c)
	//}
	//fmt.Println("=== End Core Traffic ==================================================")
	//fmt.Println("=== API Traffic =======================================================")
	//for i, c := range apiTraffic {
	//	fmt.Printf("---- %d ----------------------------\n", i)
	//	fmt.Println(c)
	//}
	//fmt.Println("=== End API Traffic ====================================================")

	// Verify Meta-Core traffic
	assert.Equal(t, 4, len(coreTraffic))

	assert.Contains(t, coreTraffic[0], "POST /device/register", "Metal-Cores register endpoint not called by metal-hammer service")
	devId := coreTraffic[0][22:strings.Index(coreTraffic[0], " HTTP")]
	rdr := &domain.MetalHammerRegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[0])), rdr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, rdr.UUID)

	assert.Contains(t, coreTraffic[1], "200 OK", "Metal-APIs register endpoint not called by meta-core service")
	dev := &models.MetalDevice{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, coreTraffic[3], fmt.Sprintf("GET /device/install/%v", devId), "Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by metal-hammer service")

	// Verify Metal-API traffic
	assert.Equal(t, 3, len(apiTraffic))

	assert.Contains(t, apiTraffic[0], "POST /device/register", "Metal-APIs register endpoint not called by metal-core service")
	mardr := &domain.MetalHammerRegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[0])), mardr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, mardr.UUID)

	assert.Contains(t, apiTraffic[1], "200 OK", "Metal-APIs register endpoint not called by meta-core service")
	dev = &models.MetalDevice{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, apiTraffic[2], fmt.Sprintf("GET /device/%v/wait", devId),
		"Either Metal-Cores install endpoint threw an error or Metal-APIs wait endpoint not called by metal-core service")
}

func extractPayload(s string) string {
	lines := strings.Split(s, "\n")
	for i := 0; i < len(lines); i++ {
		if len(strings.TrimSpace(lines[i])) == 0 {
			p := strings.TrimSpace(strings.Join(lines[i+1:], ""))
			i = strings.Index(p, "{")
			if i > 0 {
				p = p[i:]
			}
			return p
		}
	}
	return lines[len(lines)-1]
}

func runAsRoot() error {
	if mageBinary, err := exec.Command("which", "mage").CombinedOutput(); err != nil {
		panic(err)
	} else if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		bin := string(mageBinary[:len(mageBinary)-1])
		return sh.RunV("sudo", "-E", "PATH=$PATH", "/usr/bin/env", "bash", "-c", fmt.Sprintf("cd %v/../.. && %v test:e2e", wd, bin))
	}
}

func spawnMetalCoreRethinkdbAndMetalAPI() {
	readEnvFile()
	start("nsqlookupd")
	start("nsqd")
	startAndWaitFor("rethinkdb", "Server ready", 5)
	start("netbox-init-config")
	startAndWaitFor("netbox-postgres", "database system is ready to accept connections", 2)
	startAndWaitFor("netbox", "Starting gunicorn", 25)
	startAndWaitFor("netbox-nginx", "start worker process ", 5)
	startAndWaitFor("netbox-api-proxy", "Serving Flask app", 2)
	startAndWaitFor("metal-api", "start metal api", 2)
	startAndWaitFor("metal-core", "Starting metal-core", 2)
}

func start(service string) {
	if err := sh.RunV("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d", service); err != nil {
		panic(err)
	}
}

func waitFor(service, expected string, timeout int) {
	for i := 0; i < timeout; i++ {
		time.Sleep(time.Second)
		if out, err := exec.Command("docker-compose", "-f", "workflow_test.yaml", "logs", service).CombinedOutput(); err != nil {
			panic(err)
		} else if strings.Contains(string(out), expected) {
			return
		}
	}
	panic(errors.New(fmt.Sprintf("cannot fetch %v logs", service)))
}

func startAndWaitFor(service, expected string, timeout int) {
	start(service)
	waitFor(service, expected, timeout)
}

func sniffTcpPackets() {
	metalCoreHandle := createHandle(env.MetalCorePort)
	metalApiHandle := createHandle(env.MetalAPIPort)
	go traceTcpPackets(metalCoreHandle, &coreTraffic)
	go traceTcpPackets(metalApiHandle, &apiTraffic)
}

func createHandle(port int) *pcap.Handle {
	if dev, err := fetchNetworkDevice(); err != nil {
		panic(err)
	} else if handle, err := pcap.OpenLive(dev, 65536, true, pcap.BlockForever); err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(fmt.Sprintf("tcp and port %d", port)); err != nil {
		panic(err)
	} else {
		return handle
	}
}

func fetchNetworkDevice() (string, error) {
	if out, err := exec.Command("/usr/bin/env", "bash", "-c", "ip address show | grep -B2 10.0.0.1/24 | head -n1 | cut -d: -f2").CombinedOutput(); err != nil {
		return "", err
	} else {
		return strings.TrimSpace(string(out)), nil
	}
}

func traceTcpPackets(handle *pcap.Handle, traffic *[]string) {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		payload := strings.TrimSpace(string(packet.TransportLayer().LayerPayload()))
		if len(payload) > 0 {
			*traffic = append(*traffic, payload)
		}
	}
}

func readEnvFile() {
	p := properties.MustLoadFile(".env", properties.UTF8)
	if err := p.Decode(&env); err != nil {
		panic(err)
	}
}

func tearDown() {
	exec.Command("docker-compose", "-f", "workflow_test.yaml", "down").Run()
	proxy.Close()
}
