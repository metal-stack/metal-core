package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	goipmi "github.com/vmware/goipmi"
)

func openClientConnection(connection *domain.IPMIConfig) (*goipmi.Client, error) {
	conn := &goipmi.Connection{
		Hostname:  connection.Hostname,
		Port:      connection.Port,
		Username:  connection.Username,
		Password:  connection.Password,
		Interface: connection.Interface,
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
