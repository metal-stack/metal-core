package e2e

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestWorkflow(t *testing.T) {
	defer tearDown()

	// GIVEN
	spawnTestEnvironment(t)

	// WHEN
	time.Sleep(3 * time.Second)

	// THEN
	if out, err := sh.Output("docker", "logs", "metal-core-test"); err != nil {
		assert.Failf(t, "Failed to fetch docker logs from metal-core-test container", "err=%v", err)
	} else {
		expected := "http://localhost:18081/device/register"
		assert.Contains(t, out, expected, "Metal-APIs register endpoint not called by metal-core-test container")
		out = forward(out, expected)

		expected = "/device/install/230446CC-321C-11B2-A85C-AA62A1C99720"
		assert.Contains(t, out, expected, "Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by metal-hammer-test container")
		out = forward(out, expected)

		expected = "http://localhost:18081/device/230446CC-321C-11B2-A85C-AA62A1C99720/wait"
		assert.Contains(t, out, expected, "Either Metal-Cores install endpoint threw an error or Metal-APIs wait endpoint not called by metal-core-test container")
		out = forward(out, expected)
	}
}

func spawnTestEnvironment(t *testing.T) {
	if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up", "--force-recreate", "--remove-orphans", "-d"); err != nil {
		assert.Fail(t, "Failed to spin up test environment")
		os.Exit(1)
	}
	waitFor("rethinkdb-test", t)
	waitFor("metal-hammer-test", t)
	waitFor("metal-api-test", t)
	waitFor("metal-core-test", t)
}

func waitFor(container string, t *testing.T) {
	for i:=0; i<5; i++  {
		if out, err := exec.Command("docker", "ps", "-a", "-q", "-f", fmt.Sprintf("name=%v", container)).CombinedOutput(); err==nil && len(out) > 0 {
			return
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	assert.Failf(t, "Failed to spawn container %v", container)
}

func forward(out string, s string) string {
	index := strings.Index(out, s)
	if index == -1 {
		return ""
	}
	return out[index:]
}

func tearDown() {
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "kill")
}
