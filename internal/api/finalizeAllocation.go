package api

import (
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) FinalizeAllocation(machineID, consolePassword string, report *domain.Report) (*machine.FinalizeAllocationOK, error) {
	body := &models.V1MachineFinalizeAllocationRequest{
		ConsolePassword: &consolePassword,
		Primarydisk:     &report.PrimaryDisk,
		Ospartition:     &report.OSPartition,
		Initrd:          &report.Initrd,
		Cmdline:         &report.Cmdline,
		Kernel:          &report.Kernel,
		Bootloaderid:    &report.BootloaderID,
	}
	params := machine.NewFinalizeAllocationParams()
	params.ID = machineID
	params.Body = body

	ok, err := c.MachineClient.FinalizeAllocation(params, c.Auth)
	if err != nil {
		c.Log.Error("finalize failed",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
	}
	return ok, err
}
