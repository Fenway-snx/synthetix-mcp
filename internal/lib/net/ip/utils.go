package ip

import (
	"net"
	"os"
	"slices"
)

// Obtains, via a number of means the host IP address(es)
//
// Note:
// The results are processed to provide unique strings, but may contain
// entries that may refer to the same device even if the strings are
// ostensibly different.
//
// Note:
// This may employ time-expensive operations, so should only be called in
// contexts in which significant time may be expended, such as at process
// start.
func GetHostIPAddresses() ([]string, error) {

	ipAddresses := make([]string, 0, 10)

	// via net.InterfaceAddrs()
	var err1 error
	{
		var netAddresses []net.Addr
		// var ipAddresses []net.IP
		netAddresses, err1 = net.InterfaceAddrs()
		if err1 == nil {
			for _, netAddress := range netAddresses {
				if ipNet, ok := netAddress.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					s := ipNet.IP.String()
					if s != "" {
						ipAddresses = append(ipAddresses, s)
					}
				}
			}
		}
	}

	// via
	var err2 error
	{
		var hostname string
		hostname, err2 = os.Hostname()
		if err2 == nil {
			var ips []net.IP
			ips, err2 = net.LookupIP(hostname)
			if err2 == nil {
				for _, ip := range ips {
					s := ip.String()
					if s != "" {
						ipAddresses = append(ipAddresses, s)
					}
				}
			}
		}
	}

	if err1 != nil && err2 != nil {

		return nil, err1
	}

	slices.Sort(ipAddresses)

	ipAddresses = slices.Compact(ipAddresses)

	return ipAddresses, nil
}
