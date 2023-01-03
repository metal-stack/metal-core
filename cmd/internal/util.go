package internal

import (
	"fmt"
	"net"
	"net/netip"
)

func GetManagementIP(interfaceName string) (addr string, err error) {
	var (
		ief   *net.Interface
		addrs []net.Addr
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil {
		return
	}
	if addrs, err = ief.Addrs(); err != nil {
		return
	}
	for _, addr := range addrs {
		parsed, err := netip.ParseAddr(addr.String())
		if err == nil {
			return parsed.String(), nil
		}
	}
	return "", fmt.Errorf("interface %s does not have an ip address", interfaceName)
}
