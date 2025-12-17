package dialer

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Remote struct {
	host string
	dial func(network, addr string) (net.Conn, error)
}

func (remote *Remote) String() string {
	return remote.host
}

func (remote *Remote) Dial(network, addr string) (net.Conn, error) {
	return remote.dial(network, addr)
}

func NewRemote(raw string) (*Remote, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	remote := &Remote{
		host: u.Host,
	}
	switch u.Scheme {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if u.User != nil {
			auth = new(proxy.Auth)
			auth.User = u.User.Username()
			if p, ok := u.User.Password(); ok {
				auth.Password = p
			}
		}
		var dialer proxy.Dialer
		if dialer, err = proxy.SOCKS5("tcp", remote.host, auth, proxy.Direct); err != nil {
			return nil, fmt.Errorf("创建SOCKS5错误:%w", err)
		}
		remote.dial = dialer.Dial
		return remote, nil
	default:
		remote.dial = func(network, addr string) (net.Conn, error) {
			d := net.Dialer{}
			conn, err := d.Dial(network, remote.host)
			if err != nil {
				return nil, err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
			fmt.Fprintf(&b, "Proxy-Connection: Keep-Alive\r\n")
			if u.User != nil {
				token := base64.StdEncoding.EncodeToString([]byte(u.User.String()))
				fmt.Fprintf(&b, "Proxy-Authorization: Basic %s\r\n", token)
			}
			b.WriteString("\r\n")
			if _, err := conn.Write([]byte(b.String())); err != nil {
				_ = conn.Close()
				return nil, err
			}
			br := bufio.NewReader(conn)
			resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
			if err != nil {
				_ = conn.Close()
				return nil, err
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				_ = conn.Close()
				return nil, fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
			}
			return conn, nil
		}
		return remote, nil
	}
}
