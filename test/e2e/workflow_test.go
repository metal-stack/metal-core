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
	defer func() {
		tearDown()
	}()
	// GIVEN
	// Create integration test environment, i.e. spawn metal-core-test, metal-api-test and metal-hammer-test containers
	go func() {
		if _, err := sh.Output("docker-compose", "-f", "workflow_test.yaml", "up"); err != nil {
			assert.Fail(t, "Failed to spin up integration test environment")
		}
	}()

	// WHEN
	time.Sleep(2500 * time.Millisecond)

	// THEN
	if out, err := sh.Output("docker", "logs", "metal-core-test"); err != nil {
		assert.Failf(t, "Failed to fetch docker logs from metal-core-test container", "err=%v", err)
	} else {
		index := strings.Index(out, "http://localhost:8090/device/register")
		assert.NotEqual(t, -1, index, "Metal-APIs register endpoint not called by metal-core-test container")
		if index == -1 {
			return
		}
		out = out[index:]

		index = strings.Index(out, "/device/install/1234-1234-1234")
		assert.NotEqual(t, -1, index, "Either Metal-APIs register endpoint threw an error or Metal-Cores install endpoint not called by metal-hammer-test container")
		if index == -1 {
			return
		}
		out = out[index:]

		index = strings.Index(out, "http://localhost:8090/image/2")
		assert.NotEqual(t, -1, index, "Either Metal-Cores install endpoint threw an error or Metal-APIs install endpoint not called by metal-core-test container")
		if index == -1 {
			return
		}
		out = out[index:]

		index = strings.Index(out, "https://registry.maas/alpine/alpine:3.8")
		assert.NotEqual(t, -1, index, "Either Metal-APIs install endpoint threw an error or did not received expected image URL from metal-api-test container")
		if index == -1 {
			return
		}
		out = out[index:]

		index = strings.Index(out, "\"body\":\"https://blobstore.fi-ts.io/metal/images/os/ubuntu/18.04/img.tar.gz\"")
		assert.NotEqual(t, -1, index, "Did not sent expected response to metal-hammer-test container")
		if index == -1 {
			return
		}
	}
}

func tearDown() {
	sh.RunV("docker-compose", "-f", "workflow_test.yaml", "down", "--remove-orphans")
}
