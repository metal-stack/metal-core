package ipmi

import (
	"errors"
	"github.com/metal-stack/metal-core/pkg/domain"
	goipmi "github.com/vmware/goipmi"
)

const (
	defaultInterface = "lanplus"
	defaultUser      = "ADMIN"
	defaultPassword  = "ADMIN"
)

func openClientConnection(connection *domain.IPMIConfig) (*goipmi.Client, error) {
	conn := &goipmi.Connection{
		Hostname:  connection.Hostname,
		Port:      connection.Port,
		Username:  *connection.Ipmi.User,
		Password:  *connection.Ipmi.Password,
		Interface: *connection.Ipmi.Interface,
	}

	if conn.Interface == "" {
		conn.Interface = defaultInterface
	}
	if conn.Username == "" {
		conn.Username = defaultUser
	}
	if conn.Password == "" {
		conn.Password = defaultPassword
	}

	client, err := goipmi.NewClient(conn)
	if err != nil {
		return nil, err
	}

	err = client.Open()
	if err != nil {
		return nil, err
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

// ChassisIdentifyRequest per section 28.5
type ChassisIdentifyRequest struct {
	IntervalOrOff uint8
	ForceOn       uint8
}

// ChassisIdentifyResponse per section 28.5
type ChassisIdentifyResponse struct {
	goipmi.CompletionCode
}

func sendChassisIdentifyRaw(client *goipmi.Client, intervalOrOff, forceOn uint8) error {
	r := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis, // 0x00
		Command:         CommandChassisIdentifyOptions, // 0x04
		Data: &ChassisIdentifyRequest{
			IntervalOrOff: intervalOrOff,
			ForceOn:       forceOn,
		},
	}
	resp := &ChassisIdentifyResponse{}
	err := client.Send(r, resp)
	if err != nil {
		return err
	}
	if goipmi.CompletionCode(resp.CompletionCode.Code()) != goipmi.CommandCompleted {
		return errors.New(resp.Error())
	}
	return nil
}
