package lldp

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"strings"
	"time"
)

const lldpProtocol = "0x88cc"

// Client consumes lldp messages.
type Client struct {
	Source    *gopacket.PacketSource
	Handle    *pcap.Handle
	Interface *net.Interface
}

// FrameFragment holds optional TLV SysName and SysDescription fields of a real LLDP frame.
type FrameFragment struct {
	SysName        string
	SysDescription string
}

// PhoneHomeToken contains a phone-home token.
type PhoneHomeToken struct {
	EventType endpoint.ProvisioningEventType
	MachineID string
	Payload   string
}

// NewClient creates a new LLDP client.
func NewClient(interfaceName string) (*Client, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to lookup interface:%s", interfaceName)
	}

	zapup.MustRootLogger().Info("lldp",
		zap.String("listen on interface", iface.Name),
	)

	handle, err := pcap.OpenLive(iface.Name, 65536, true, 5*time.Second)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open interface:%s in promiscuous mode", iface.Name)
	}

	// filter only LLDP packages
	bpfFilter := fmt.Sprintf("ether proto %s", lldpProtocol)
	err = handle.SetBPFFilter(bpfFilter)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to filter LLDP ethernet traffic %s on interface:%s", lldpProtocol, iface.Name)
	}

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	return &Client{
		Source:    src,
		Handle:    handle,
		Interface: iface,
	}, nil
}

// CatchPackages searches on the configured interface for LLDP packages and
// pushes the optional TLV SysName and SysDescription fields of each
// found LLDP package into the given channel.
func (l *Client) CatchPackages(frameFragmentChannel chan FrameFragment) {
	defer func() {
		close(frameFragmentChannel)
		l.Close()
	}()

	for {
		for packet := range l.Source.Packets() {
			switch packet.LinkLayer().LayerType() {
			case layers.LayerTypeEthernet:
				ff := FrameFragment{}
				for _, layer := range packet.Layers() {
					layerType := layer.LayerType()
					switch layerType {
					case layers.LayerTypeLinkLayerDiscoveryInfo:
						info := layer.(*layers.LinkLayerDiscoveryInfo)
						ff.SysName = info.SysName
						ff.SysDescription = info.SysDescription
						frameFragmentChannel <- ff
					}
				}
			}
		}
	}
}

// Close the LLDP client
func (l *Client) Close() {
	l.Handle.Close()
}

// ExtractPhoneHomeToken extracts the machineID and payload of the given LLDP frame fragment.
// An error will be returned if the frame fragment does not contain a phone-home token.
func (l *Client) ExtractPhoneHomeToken(frameFragment *FrameFragment) (*PhoneHomeToken, error) {
	token := &PhoneHomeToken{
		MachineID: frameFragment.SysName,
		Payload:   frameFragment.SysDescription,
	}

	if strings.Contains(token.Payload, "provisioning") {
		token.EventType = endpoint.ProvisioningEventPhonedHome
		return token, nil
	}

	if strings.Contains(token.Payload, "metal-hammer") || strings.Contains(token.Payload, "waiting since") {
		token.EventType = endpoint.ProvisioningEventWaiting
		return token, nil
	}

	return nil, errors.New("LLDP package contains no phone-home token")
}
