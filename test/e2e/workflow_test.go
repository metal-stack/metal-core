package e2e

import (
	"github.com/magefile/mage/sh"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestWorkflow(t *testing.T) {
	tearDown()
	defer kill()

	// GIVEN
	// Create integration test environment, i.e. spawn metal-core-test, metal-api-test and metal-hammer-test containers
	go func() {
		if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up"); err != nil {
			assert.Fail(t, "Failed to spin up integration test environment")
		}
	}()

	// WHEN
	time.Sleep(5000 * time.Millisecond)

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

func forward(out string, s string) string {
	index := strings.Index(out, s)
	if index == -1 {
		return ""
	}
	return out[index:]
}

func kill() {
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "kill")
}

func tearDown() {
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "down", "--remove-orphans")
}
