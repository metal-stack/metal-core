package frr

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/metal-stack/metal-go/api/models"
)

type Vrf struct {
	VrfID   int
	VrfName string
	Ports   Ports
}

type Port struct {
	Hostname              string `json:"hostname"`
	PeerGroup             string `json:"peerGroup"`
	BgpState              string `json:"bgpState"`
	BgpTimerUpEstablished int64  `json:"bgpTimerUpEstablishedEpoch"`

	AddressFamilyInfo struct {
		IPv4UnicastCumulus struct {
			SentPrefixCounter     int64 `json:"sentPrefixCounter"`
			AcceptedPrefixCounter int64 `json:"acceptedPrefixCounter"`
		} `json:"IPv4 Unicast"`
		IPv6UnicastCumulus struct {
			SentPrefixCounter     int64 `json:"sentPrefixCounter"`
			AcceptedPrefixCounter int64 `json:"acceptedPrefixCounter"`
		} `json:"IPv6 Unicast"`
		IPv4UnicastSonic struct {
			SentPrefixCounter     int64 `json:"sentPrefixCounter"`
			AcceptedPrefixCounter int64 `json:"acceptedPrefixCounter"`
		} `json:"ipv4Unicast"`
		IPv6UnicastSonic struct {
			SentPrefixCounter     int64 `json:"sentPrefixCounter"`
			AcceptedPrefixCounter int64 `json:"acceptedPrefixCounter"`
		} `json:"ipv6Unicast"`
	} `json:"addressFamilyInfo"`
}

type Vrfs map[string]Vrf
type Ports map[string]Port

func GetBGPStates(filepath string) (map[string]models.V1SwitchBGPPortState, error) {

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("error getting file info for %s: %w", filepath, err)
	}

	if time.Since(fileInfo.ModTime()) > time.Hour {
		return nil, fmt.Errorf("file %s is too old", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening frr bgp state json file %s: %w", filepath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading frr bgp state json file %s: %w", filepath, err)
	}

	var tempData map[string]map[string]json.RawMessage
	if err := json.Unmarshal(byteValue, &tempData); err != nil {
		return nil, fmt.Errorf("error unmarshalling bgp vrf all neigh output: %w", err)
	}

	bgpstates := make(map[string]models.V1SwitchBGPPortState)
	for _, vrfData := range tempData {

		var VrfName string
		if err := json.Unmarshal(vrfData["vrfName"], &VrfName); err != nil {
			return nil, fmt.Errorf("error parsing vrfName: %w", err)
		}

		for key, value := range vrfData {
			if key == "vrfId" || key == "vrfName" {
				continue
			}
			var port Port
			if err := json.Unmarshal(value, &port); err != nil {
				return nil, fmt.Errorf("error parsing port info for %s: %w", key, err)
			}
			bgptimerup := port.BgpTimerUpEstablished
			sentPrefixCounter := port.AddressFamilyInfo.IPv4UnicastCumulus.SentPrefixCounter +
				port.AddressFamilyInfo.IPv6UnicastCumulus.SentPrefixCounter +
				port.AddressFamilyInfo.IPv4UnicastSonic.SentPrefixCounter +
				port.AddressFamilyInfo.IPv6UnicastSonic.SentPrefixCounter

			acceptedPrefixCounter := port.AddressFamilyInfo.IPv4UnicastCumulus.AcceptedPrefixCounter +
				port.AddressFamilyInfo.IPv6UnicastCumulus.AcceptedPrefixCounter +
				port.AddressFamilyInfo.IPv4UnicastSonic.AcceptedPrefixCounter +
				port.AddressFamilyInfo.IPv6UnicastSonic.AcceptedPrefixCounter

			bgpstates[key] = models.V1SwitchBGPPortState{
				Neighbor:              &port.Hostname,
				PeerGroup:             &port.PeerGroup,
				BgpState:              &port.BgpState,
				BgpTimerUpEstablished: &bgptimerup,
				VrfName:               &VrfName,
				SentPrefixCounter:     &sentPrefixCounter,
				AcceptedPrefixCounter: &acceptedPrefixCounter,
			}
		}
	}

	return bgpstates, nil
}
