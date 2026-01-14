package dialer

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	sync2 "github.com/yydsqu/tools/sync"
	"io"
	"net"
	"strings"
	"time"
)

var (
	DefaultTarget          = "cloudflare.com:80"
	IPInfoTarget           = "https://cloudflare.com/cdn-cgi/trace"
	IPIFYV4Target          = "https://api.ipify.org/"
	IPIFYV6Target          = "https://api6.ipify.org"
	DefaultVirtualPrefixes = []string{"lo", "docker", "br-", "veth", "tun", "tap", "virbr", "VMware"}
)

func ParseTrace(r io.Reader) map[string]string {
	m := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err = sync2.GroupGenericWithContext(ctx, ips, func(ctx context.Context, ip net.IP) (net.IP, error) {
		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP:   ip,
				Port: 0,
			},
		}
		conn, err2 := dialer.DialContext(ctx, "tcp4", DefaultTarget)
		if err2 != nil {
			return nil, err2
		}
		defer conn.Close()
		return ip, nil
	})
	// IP信息去重信息
	if len(ips) == 0 {
		return nil, cmp.Or(err, fmt.Errorf("没有可用的IP"))
	}
	return ips, nil
}
