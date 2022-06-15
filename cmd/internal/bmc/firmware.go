package bmc

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/pkg/api"
	metalgo "github.com/metal-stack/metal-go"
)

func (b *BMCService) UpdateFirmware(outBand hal.OutBand, event *MachineEvent) {
	kind := metalgo.FirmwareKind(event.Cmd.Params[0])
	revision := event.Cmd.Params[1]
	// description := event.Cmd.Params[2]
	s3Cfg := &api.S3Config{
		Url:            event.Cmd.Params[3],
		Key:            event.Cmd.Params[4],
		Secret:         event.Cmd.Params[5],
		FirmwareBucket: event.Cmd.Params[6],
	}
	board := event.Cmd.IPMI.Fru.BoardPartNumber
	switch kind {
	case metalgo.Bios:
		go func() {
			err := outBand.UpdateBIOS(board, revision, s3Cfg)
			if err != nil {
				b.log.Errorw("updatebios", "error", err)
			}
		}()
	case metalgo.Bmc:
		go func() {
			err := outBand.UpdateBMC(board, revision, s3Cfg)
			if err != nil {
				b.log.Errorw("updatebmc", "error", err)
			}
		}()
	default:
		b.log.Errorw("unknown firmware kind", "topic", b.machineTopic, "channel", "core", "firmware kind", string(kind), "event", event)
	}
}
