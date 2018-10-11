package e2e

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/tcpproxy"
	"github.com/magefile/mage/sh"
	"github.com/magiconair/properties"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type Env struct {
	MetalAPIPort             int    `properties:"METAL_API_PORT"`
	RethinkdbPort            int    `properties:"RETHINKDB_PORT"`
	MetalHammerContainerName string `properties:"METAL_HAMMER_CONTAINER_NAME"`
	MetalCoreContainerName   string `properties:"METAL_CORE_CONTAINER_NAME"`
	MetalAPIContainerName    string `properties:"METAL_API_CONTAINER_NAME"`
	RethinkdbContainerName   string `properties:"RETHINKDB_CONTAINER_NAME"`
}

var (
	env   Env
	proxy tcpproxy.Proxy
)

func TestWorkflow(t *testing.T) {
	defer tearDown()

	// GIVEN
	spawnTestEnvironment(t)

	// WHEN
	runMetalHammer(t)

	// THEN
	if out, err := sh.Output("docker", "logs", "metal-core-test"); err != nil {
		panic(err)
	} else {
		expected := "http://localhost:18081/device/register"
		assert.Contains(t, out, expected, fmt.Sprintf("Metal-APIs register endpoint not called by %v container", env.MetalCoreContainerName))
		out = forward(out, expected)

		expected = "/device/install/"
		assert.Contains(t, out, expected, fmt.Sprintf("Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by %v container", env.MetalHammerContainerName))
		out = forward(out, expected)

		expected = "/wait"
		assert.Contains(t, out, expected, fmt.Sprintf("Either Metal-Cores install endpoint threw an error or Metal-APIs wait endpoint not called by %v container", env.MetalCoreContainerName))
		out = forward(out, expected)
	}
}

func spawnTestEnvironment(t *testing.T) {
	readEnvFile(t)
	removeMetalHammerContainer()
	go sniffTCPPackets()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d"); err != nil {
		panic(err)
	}
}

func sniffTCPPackets() {
	if handle, err := pcap.OpenLive("lo", 1800, false, pcap.BlockForever); err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(fmt.Sprintf("tcp and port %d", env.MetalAPIPort)); err != nil {
		panic(err)
	} else {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		i := 1
		for packet := range packetSource.Packets() {
			payload := string(packet.TransportLayer().LayerPayload())
			if len(payload) > 0 {
				fmt.Printf("%d:\n", i)
				i++
				fmt.Println(payload)
			}
		}
	}
}

func runMetalHammer(t *testing.T) {
	removeMetalHammerContainer()
	time.Sleep(3 * time.Second)
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "run", "-d", "--name", env.MetalHammerContainerName, "--entrypoint", "/metal-hammer", "metal-hammer"); err != nil {
		panic(err)
	}
}

func readEnvFile(t *testing.T) {
	p := properties.MustLoadFile(".env", properties.UTF8)
	if err := p.Decode(&env); err != nil {
		panic(err)
	}
}

func removeMetalHammerContainer() {
	exec.Command("docker", "rm", "-f", env.MetalHammerContainerName).Run()
}

func forward(out string, s string) string {
	index := strings.Index(out, s)
	if index == -1 {
		return ""
	}
	return out[index:]
}

func tearDown() {
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "down")
	proxy.Close()
}
