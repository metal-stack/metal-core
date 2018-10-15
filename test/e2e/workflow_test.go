package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
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
	spawnTestEnvironment()

	// WHEN
	runMetalHammer()

	// THEN
	// Verify core traffic
	assert.Equal(t, 3, len(coreTraffic))

	assert.Contains(t, coreTraffic[0], "POST /device/register", fmt.Sprintf("Metal-Cores register endpoint not called by %v container", env.MetalHammerContainerName))
	devId := coreTraffic[0][22:strings.Index(coreTraffic[0], " HTTP")]
	rdr := &domain.RegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[0])), rdr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, rdr.UUID)

	assert.Contains(t, coreTraffic[1], "200 OK", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	dev := &domain.Device{}
	if err := json.Unmarshal([]byte(extractPayload(coreTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, coreTraffic[2], fmt.Sprintf("GET /device/install/%v", devId), fmt.Sprintf("Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by %v container", env.MetalHammerContainerName))

	// Verify api traffic
	assert.Equal(t, 3, len(apiTraffic))

	assert.Contains(t, apiTraffic[0], "POST /device/register", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	mardr := &domain.MetalApiRegisterDeviceRequest{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[0])), mardr); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, mardr.UUID)

	assert.Contains(t, apiTraffic[1], "200 OK", fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
	dev = &domain.Device{}
	if err := json.Unmarshal([]byte(extractPayload(apiTraffic[1])), dev); err != nil {
		panic(err)
	}
	assert.Equal(t, devId, dev.ID)

	assert.Contains(t, apiTraffic[2], fmt.Sprintf("GET /device/%v/wait", devId), fmt.Sprintf("Either Metal-Cores install endpoint threw an error or Metal-APIs wait endpoint not called by %v container", env.MetalCoreContainerName))
}

func extractPayload(s string) string {
	lines := strings.Split(s, "\n")
	for i := 0; i < len(lines); i++ {
		if len(strings.TrimSpace(lines[i])) == 0 {
			return strings.Join(lines[i+1:], "")
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

func spawnTestEnvironment() {
	readEnvFile()
	removeMetalHammerContainer()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d"); err != nil {
		panic(err)
	}
}

func runMetalHammer() {
	removeMetalHammerContainer()
	time.Sleep(3 * time.Second)
	waitForMetalApiContainer()
	sniffTcpPackets()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "run", "-d", "--name", env.MetalHammerContainerName, "--entrypoint", "/metal-hammer", "metal-hammer"); err != nil {
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
	if handle, err := pcap.OpenLive("lo", 1600, false, pcap.BlockForever); err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(fmt.Sprintf("tcp and port %d", port)); err != nil {
		panic(err)
	} else {
		return handle
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
	for i := 0; i < 10; i++ {
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
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "down")
	proxy.Close()
}
