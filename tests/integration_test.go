package tests

import (
	"github.com/magefile/mage/sh"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestContainerInteraction(t *testing.T) {
	tearDown()
	defer func() {
		tearDown()
	}()
	// GIVEN
	// Create integration test environment, i.e. spawn metal-core, metal-api and discover containers
	go func() {
		if _, err := sh.Output("docker-compose", "-f", "integration_test.yaml", "up"); err != nil {
			assert.Fail(t, "Failed to spin up integration test environment")
		}
	}()

	// WHEN
	time.Sleep(2500 * time.Millisecond)

	// THEN
	if out, err := sh.Output("docker", "logs", "metal-core"); err != nil {
		assert.Fail(t, "Failed to fetch docker logs from metal-core container")
	} else {
		index := strings.Index(out, "http://localhost:8090/device/register")
		assert.NotEqual(t, -1, index, "Metal-APIs register endpoint not called by metal-core container")
		out = out[index:]

		index = strings.Index(out, "/device/install/1234-1234-1234")
		assert.NotEqual(t, -1, index, "Metal-Cores install endpoint not called by discover container")
		out = out[index:]

		index = strings.Index(out, "http://localhost:8090/image/2")
		assert.NotEqual(t, -1, index, "Metal-APIs install endpoint not called by metal-core container")
		out = out[index:]

		index = strings.Index(out, "https://registry.maas/alpine/alpine:3.8")
		assert.NotEqual(t, -1, index, "Did not received expected image URL from metal-api container")
		out = out[index:]

		index = strings.Index(out, "\"body\":\"https://blobstore.fi-ts.io/metal/images/os/ubuntu/18.04/img.tar.gz\"")
		assert.NotEqual(t, -1, index, "Did not sent expected response to discover container")
	}
}

func tearDown() {
	sh.Run("docker-compose", "-f", "integration_test.yaml", "down", "--remove-orphans")
}
