package ipmi

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

func openClientConnection(connection *domain.IpmiConnection) (*goipmi.Client, error) {
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

func PowerOn(connection *domain.IpmiConnection) error {
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

func PowerOff(connection *domain.IpmiConnection) error {
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

func SetBootDevPxe(connection *domain.IpmiConnection) error {
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

func SetBootDevHd(connection *domain.IpmiConnection) error {
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
