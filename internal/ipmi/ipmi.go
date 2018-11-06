package ipmi

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

type IpmiConnection struct {
	Hostname  string
	Interface string
	Port      int
	Username  string
	Password  string
}

func openClientConnection(connection *IpmiConnection) (*goipmi.Client, error) {
	goipmiConnection := &goipmi.Connection{
		Hostname:  connection.Hostname,
		Port:      connection.Port,
		Username:  connection.Username,
		Password:  connection.Password,
		Interface: connection.Interface,
	}
	client, err := goipmi.NewClient(goipmiConnection)
	if err != nil {
		return client, err
	}
	err = client.Open()
	if err != nil {
		return client, err
	}
	return client, nil
}

// FIXME: This is just for the POC
// func testIpmi() {
// 	gateway, err := gateway.DiscoverGateway()
// 	if err != nil {
// 		log.Error("Unable to determine gateway for reaching out to ipmi client: ", err)
// 		return
// 	} else {
// 		log.Infof("Reaching out to ipmi client through gateway: %s", gateway.String())
// 	}
// 	connection := &ipmi.IpmiConnection{
// 		Hostname:  gateway.String(),
// 		Interface: "lanplus",
// 		Port:      6230,
// 		Username:  "vagrant",
// 		Password:  "vagrant",
// 	}
// 	ipmi.SetBootDevHd(connection)
// }

func PowerOn(connection *IpmiConnection) error {
	client, err := openClientConnection(connection)
	if err != nil {
		return err
	}
	zapup.MustRootLogger().Info("Powering up",
		zap.String("hostname", connection.Hostname),
	)
	err = client.Control(goipmi.ControlPowerUp)
	if err != nil {
		return err
	}
	return nil
}

func PowerOff(connection *IpmiConnection) error {
	client, err := openClientConnection(connection)
	if err != nil {
		return err
	}
	zapup.MustRootLogger().Info("Powering off",
		zap.String("hostname", connection.Hostname),
	)
	err = client.Control(goipmi.ControlPowerDown)
	if err != nil {
		return err
	}
	return nil
}

func SetBootDevPxe(connection *IpmiConnection) error {
	client, err := openClientConnection(connection)
	if err != nil {
		return err
	}
	zapup.MustRootLogger().Info("Setting boot device to PXE boot",
		zap.String("hostname", connection.Hostname),
	)
	err = client.SetBootDevice(goipmi.BootDevicePxe)
	if err != nil {
		return err
	}
	return nil
}

func SetBootDevHd(connection *IpmiConnection) error {
	client, err := openClientConnection(connection)
	if err != nil {
		return err
	}
	zapup.MustRootLogger().Info("Setting boot device to HD boot",
		zap.String("hostname", connection.Hostname),
	)
	err = client.SetBootDevice(goipmi.BootDeviceDisk)
	if err != nil {
		return err
	}
	return nil
}
