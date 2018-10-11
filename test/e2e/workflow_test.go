package e2e

import (
	"fmt"
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
	MetalAPIPort             int    `properties:"METAL_API_PORT"`
	RethinkdbPort            int    `properties:"RETHINKDB_PORT"`
	MetalHammerContainerName string `properties:"METAL_HAMMER_CONTAINER_NAME"`
	MetalCoreContainerName   string `properties:"METAL_CORE_CONTAINER_NAME"`
	MetalAPIContainerName    string `properties:"METAL_API_CONTAINER_NAME"`
	RethinkdbContainerName   string `properties:"RETHINKDB_CONTAINER_NAME"`
}

var env Env

func TestWorkflow(t *testing.T) {
	defer tearDown()

	// GIVEN
	spawnTestEnvironment(t)

	// WHEN
	runMetalHammer(t)

	// THEN
	if out, err := sh.Output("docker", "logs", "metal-core-test"); err != nil {
		assert.Failf(t, "Failed to fetch docker logs", "container=%v, err=%v", env.MetalHammerContainerName, err)
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
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d"); err != nil {
		assert.Fail(t, "Failed to spin up test environment")
		fmt.Println(err)
		os.Exit(1)
	}
}

func runMetalHammer(t *testing.T) {
	time.Sleep(3 * time.Second)
	removeMetalHammerContainer()
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "run", "-d", "--name", env.MetalHammerContainerName, "--entrypoint", "/metal-hammer", "metal-hammer"); err != nil {
		assert.Fail(t, "Failed to run metal-hammer container")
		fmt.Println(err)
		os.Exit(1)
	}
}

func readEnvFile(t *testing.T) {
	p := properties.MustLoadFile(".env", properties.UTF8)
	if err := p.Decode(&env); err != nil {
		assert.Fail(t, err.Error())
		fmt.Println(err)
		os.Exit(1)
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
}
