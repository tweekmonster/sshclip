package platform

import (
	"net"
	"strings"
)

// LocalIsServer tests if the host is the local computer.
func LocalIsServer(host string) (bool, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false, err
	}

	for _, ip := range ips {
		if net.IPv4zero.Equal(ip) || net.IPv6zero.Equal(ip) {
			return true, nil
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, err
	}

	for _, addr := range addrs {
		ipStr := addr.String()
		if i := strings.LastIndex(ipStr, "/"); i != -1 {
			ipStr = ipStr[:i]
		}
		ip := net.ParseIP(ipStr)

		for _, hostIP := range ips {
			if ip.Equal(hostIP) {
				return true, nil
			}
		}
	}

	return false, nil
}
