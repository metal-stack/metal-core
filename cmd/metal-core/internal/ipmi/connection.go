package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	goipmi "github.com/vmware/goipmi"
)

func openClientConnection(connection *domain.IPMIConfig) (*goipmi.Client, error) {
	conn := &goipmi.Connection{
		Hostname:  connection.Hostname,
		Port:      connection.Port,
		Username:  *connection.Ipmi.User,
		Password:  *connection.Ipmi.Password,
		Interface: *connection.Ipmi.Interface,
	}

	client, err := goipmi.NewClient(conn)
	if err != nil {
		return client, err
	}

	err = client.Open()
	if err != nil {
		return client, err
	}
	return client, nil
}

func sendSystemBootRaw(client *goipmi.Client, param uint8, data ...uint8) error {
	r := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,      // 0x00
		Command:         goipmi.CommandSetSystemBootOptions, // 0x08
		Data: &goipmi.SetSystemBootOptionsRequest{
			Param: param,
			Data:  data,
		},
	}
	return client.Send(r, &goipmi.SetSystemBootOptionsResponse{})
}

const (
	CommandChassisIdentifyOptions = goipmi.Command(0x04)
)

// SetChassisIdentifyOptionsRequest per section 28.5:
// https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
type SetChassisIdentifyOptionsRequest struct {
	Data []uint8
}

// SetChassisIdentifyOptionsResponse per section 28.5:
// https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
type SetChassisIdentifyOptionsResponse struct {
	goipmi.CompletionCode
}

func sendChassisIdentifyRaw(client *goipmi.Client, data ...uint8) error {
	r := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis, // 0x00
		Command:         CommandChassisIdentifyOptions, // 0x04
		Data: &SetChassisIdentifyOptionsRequest{
			Data: data,
		},
	}
	return client.Send(r, &SetChassisIdentifyOptionsResponse{})
}
