package dialer

import (
	"context"
	"fmt"
	sync2 "github.com/yydsqu/tools/sync"
	"net"
	"strings"
	"time"
)

var (
	DefaultTarget          = "1.1.1.1:443"
	DefaultVirtualPrefixes = []string{"lo", "docker", "br-", "veth", "tun", "tap", "virbr", "VMware"}
)

func isVirtualInterface(iface string) bool {
	for _, prefix := range DefaultVirtualPrefixes {
		if strings.HasPrefix(iface, prefix) {
			return true
		}
	}
	return false
}

func LoadLocalIPV4() ([]net.IP, error) {
	var (
		IPS []net.IP
	)
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("加载网卡失败: %v", err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}
		var addrs []net.Addr
		if addrs, err = iface.Addrs(); err != nil {
			return nil, fmt.Errorf("加载网卡[%s]地址失败: %w", iface.Name, err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4.IsLoopback() || ip4.IsLinkLocalUnicast() || !ip4.IsGlobalUnicast() {
				continue
			}
			IPS = append(IPS, ip)
		}
	}
	if len(IPS) == 0 {
		return nil, fmt.Errorf("没有可用的IP地址")
	}
	return IPS, nil
}

func LoadReachableIPV4() ([]net.IP, error) {
	ips, err := LoadLocalIPV4()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return sync2.GroupGenericWithContext(ctx, ips, func(ctx context.Context, ip net.IP) (net.IP, error) {
		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: ip,
			},
		}
		conn, err := dialer.DialContext(ctx, "tcp", DefaultTarget)
		if err != nil {
			return nil, fmt.Errorf("IP:%s不可用", ip.String())
		}
		defer conn.Close()
		return ip, nil
	})
}
