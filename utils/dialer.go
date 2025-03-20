package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/yydsqu/tools/log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	DefaultTarget          = "1.1.1.1:80"
	DefaultVirtualPrefixes = []string{"lo", "docker", "br-", "veth", "tun", "tap", "virbr", "VMware"}
)

type Dial interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	Dial(network, address string) (net.Conn, error)
}

type Local struct {
	ip   net.IP
	dial Dial
}

func (l *Local) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return l.dial.DialContext(ctx, network, address)
}

func (l *Local) Dial(network, address string) (net.Conn, error) {
	return l.dial.Dial(network, address)
}

func (l *Local) String() string {
	return l.ip.String()
}

func isVirtualInterface(iface string) bool {
	for _, prefix := range DefaultVirtualPrefixes {
		if strings.HasPrefix(iface, prefix) {
			return true
		}
	}
	return false
}

func LoadLocalDialer(IsPrivate bool) ([]Dial, error) {
	var (
		wg         sync.WaitGroup
		mutex      sync.Mutex
		interfaces []net.Interface
		addrs      []net.Addr
		available  []Dial
		err        error
	)

	if interfaces, err = net.Interfaces(); err != nil {
		return nil, err
	}

	for _, n := range interfaces {
		if n.Flags&net.FlagUp == 0 || n.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(n.Name) {
			continue
		}
		if addrs, err = n.Addrs(); err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if len(ip.To4()) != net.IPv4len || (!IsPrivate && ip.IsPrivate()) {
				continue
			}
			wg.Add(1)
			go func(ip net.IP) {
				defer wg.Done()
				dialer := &net.Dialer{
					LocalAddr: &net.TCPAddr{IP: ip},
					Control:   Control,
				}
				conn, err := dialer.Dial("tcp", DefaultTarget)
				if err != nil {
					log.Trace("test dialer failed", "ip", ip.String())
					return
				}
				defer conn.Close()
				mutex.Lock()
				defer mutex.Unlock()
				log.Trace("test dialer successful", "ip", ip.String())
				available = append(available, &Local{
					ip:   ip,
					dial: dialer,
				})
			}(ip)
		}
	}
	wg.Wait()
	if len(available) == 0 {
		return nil, fmt.Errorf("no valid network interfaces found")
	}

	log.Trace("load available ip successful", "len", len(available))

	return available, nil
}

type Remote struct {
	proxyAddr string
}

func (p *Remote) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, p.proxyAddr)
	if err != nil {
		return nil, err
	}
	connectRequest := "CONNECT " + addr + " HTTP/1.1\r\nHost: " + addr + "\r\n\r\n"
	_, err = conn.Write([]byte(connectRequest))
	if err != nil {
		conn.Close()
		return nil, err
	}
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		return nil, err
	}
	response := string(buffer[:n])
	if response[:12] != "HTTP/1.1 200" {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (p *Remote) Dial(network, addr string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.Dial(network, p.proxyAddr)
	if err != nil {
		return nil, err
	}
	connectRequest := "CONNECT " + addr + " HTTP/1.1\r\nHost: " + addr + "\r\n\r\n"
	_, err = conn.Write([]byte(connectRequest))
	if err != nil {
		conn.Close()
		return nil, err
	}
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		return nil, err
	}
	response := string(buffer[:n])
	if response[:12] != "HTTP/1.1 200" {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (p *Remote) String() string {
	return p.proxyAddr
}

func LoadRemoteDialer(proxyUrls ...string) ([]Dial, error) {
	var remotes []Dial
	for _, proxyUrl := range proxyUrls {
		remotes = append(remotes, &Remote{
			proxyAddr: proxyUrl,
		})
	}
	return remotes, nil
}

type PollingClient struct {
	clients []*http.Client
	index   atomic.Int64
}

func (c *PollingClient) CloseIdleConnections() {
	for _, client := range c.clients {
		client.CloseIdleConnections()
	}
}

func (c *PollingClient) Do(request *http.Request) (*http.Response, error) {
	return c.clients[c.index.Add(1)%int64(len(c.clients))].Do(request)
}

func (c *PollingClient) Get(url string) (*http.Response, error) {
	return c.clients[c.index.Add(1)%int64(len(c.clients))].Get(url)
}

func LoadPollingClient(IsPrivate bool, proxyUrls ...string) *PollingClient {
	cli := &PollingClient{
		clients: nil,
		index:   atomic.Int64{},
	}
	local, err := LoadLocalDialer(IsPrivate)
	if err != nil {
		cli.clients = append(cli.clients, http.DefaultClient)
		return cli
	}
	remote, err := LoadRemoteDialer(proxyUrls...)
	if len(local)+len(remote) == 0 {
		cli.clients = append(cli.clients, http.DefaultClient)
		return cli
	}
	// 加载代理配置
	cli.clients = make([]*http.Client, len(local))
	for i, d := range local {
		cli.clients[i] = &http.Client{
			Transport: &http.Transport{
				DialContext: d.DialContext,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	for i, d := range remote {
		cli.clients[i] = &http.Client{
			Transport: &http.Transport{
				DialContext: d.DialContext,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: 180 * time.Second,
		}
	}
	return cli
}
