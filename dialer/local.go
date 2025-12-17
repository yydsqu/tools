package dialer

import (
	"golang.org/x/net/proxy"
	"net"
)

var (
	Direct = proxy.Direct
)

type LocalDialer struct {
	*net.Dialer
	ip net.IP
}

func (dialer LocalDialer) String() string {
	return dialer.ip.String()
}

func LoadLocalDialer() ([]*LocalDialer, error) {
	ipv4, err := LoadReachableIPV4()
	if err != nil {
		return nil, err
	}
	dialers := make([]*LocalDialer, 0, len(ipv4))
	for _, ip := range ipv4 {
		dialers = append(dialers, &LocalDialer{
			Dialer: &net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP: ip,
				},
			},
			ip: ip,
		})
	}
	return dialers, nil
}
