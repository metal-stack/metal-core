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
	MetalCorePort            int    `properties:"METAL_CORE_PORT"`
	MetalAPIPort             int    `properties:"METAL_API_PORT"`
	RethinkdbPort            int    `properties:"RETHINKDB_PORT"`
	MetalHammerContainerName string `properties:"METAL_HAMMER_CONTAINER_NAME"`
	MetalCoreContainerName   string `properties:"METAL_CORE_CONTAINER_NAME"`
	MetalAPIContainerName    string `properties:"METAL_API_CONTAINER_NAME"`
	RethinkdbContainerName   string `properties:"RETHINKDB_CONTAINER_NAME"`
}

var (
	env         Env
	proxy       tcpproxy.Proxy
	coreTraffic []string
	apiTraffic  []string
)

func TestWorkflow(t *testing.T) {
	if os.Getegid() != 0 {
		runAsRoot()
		os.Exit(0)
	}
	defer tearDown()

	// GIVEN
	spawnMetalCoreRethinkdbAndMetalAPI()

	// WHEN
	runMetalHammer()

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

	assert.Contains(t, coreTraffic[0], "POST /device/register", fmt.Sprintf("Metal-Cores register endpoint not called by %v container", env.MetalHammerContainerName))
	devId := coreTraffic[0][22:strings.Index(coreTraffic[0], " HTTP")]
	rdr := &domain.MetalHammerRegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[0])), rdr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, rdr.UUID)

	assert.Contains(t, coreTraffic[1], "200 OK", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	dev := &models.MetalDevice{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, coreTraffic[3], fmt.Sprintf("GET /device/install/%v", devId), fmt.Sprintf("Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by %v container", env.MetalHammerContainerName))

	// Verify Metal-API traffic
	assert.Equal(t, 3, len(apiTraffic))

	assert.Contains(t, apiTraffic[0], "POST /device/register", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	mardr := &domain.MetalHammerRegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[0])), mardr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, mardr.UUID)

	assert.Contains(t, apiTraffic[1], "200 OK", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	dev = &models.MetalDevice{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, apiTraffic[2], fmt.Sprintf("GET /device/%v/wait", devId),
		fmt.Sprintf("Either Metal-Cores install endpoint threw an error or Metal-APIs wait endpoint not called by %v container", env.MetalCoreContainerName))
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

func runAsRoot() {
	if mageBinary, err := sh.Output("which", "mage"); err != nil {
		panic(err)
	} else if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		sh.RunV("sudo", "-E", "PATH=$PATH", "bash", "-c", fmt.Sprintf("cd %v/../.. && %v test:e2e", wd, mageBinary))
	}
}

func spawnMetalCoreRethinkdbAndMetalAPI() {
	readEnvFile()
	removeMetalHammerContainer()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d"); err != nil {
		panic(err)
	}
}

func runMetalHammer() {
	removeMetalHammerContainer()
	time.Sleep(15 * time.Second)
	waitForMetalApiContainer()
	sniffTcpPackets()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "run",
		"-d", "--name", env.MetalHammerContainerName, "metal-hammer"); err != nil {
		panic(err)
	}
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

func waitForMetalApiContainer() {
	for i := 0; i < 4; i++ {
		time.Sleep(time.Second)
		if out, err := sh.Output("docker", "logs", env.MetalAPIContainerName); err != nil {
			panic(err)
		} else if strings.Contains(out, "Rethinkstore connected") && strings.Contains(out, "start metal api") {
			return
		}
	}
	panic(errors.New("cannot fetch Metal-API logs"))
}

func readEnvFile() {
	p := properties.MustLoadFile(".env", properties.UTF8)
	if err := p.Decode(&env); err != nil {
		panic(err)
	}
}

func removeMetalHammerContainer() {
	exec.Command("docker", "rm", "-f", env.MetalHammerContainerName).Run()
}

func tearDown() {
	//sh.RunV("docker-compose", "-f", "workflow_test.yaml", "down")
	proxy.Close()
}
