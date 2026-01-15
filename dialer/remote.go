package dialer

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Conn func(ctx context.Context, network, address string) (net.Conn, error)

type Remote struct {
	uri  *url.URL
	dial func(ctx context.Context, network, address string) (net.Conn, error)
}

func (remote *Remote) String() string {
	return remote.uri.String()
}

func (remote *Remote) Dial(network, addr string) (net.Conn, error) {
	return remote.DialContext(context.Background(), network, addr)
}

func (remote *Remote) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return remote.dial(ctx, network, address)
}

func NewSocks5(uri *url.URL) (Conn, error) {
	var (
		auth   *proxy.Auth
		dialer proxy.Dialer
		err    error
	)
	if uri.User != nil {
		auth = new(proxy.Auth)
		auth.User = uri.User.Username()
		if p, ok := uri.User.Password(); ok {
			auth.Password = p
		}
	}
	if dialer, err = proxy.SOCKS5("tcp", uri.Host, auth, proxy.Direct); err != nil {
		return nil, fmt.Errorf("创建SOCKS5错误:%w", err)
	}
	if f, ok := dialer.(proxy.ContextDialer); ok {
		return f.DialContext, nil
	}
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		return dialer.Dial(network, address)
	}, err
}

func NewHttp(uri *url.URL) (Conn, error) {
	dialer := net.Dialer{}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, uri.Host)
		if err != nil {
			return nil, fmt.Errorf("拨号代理%s失败: %w", uri.Host, err)
		}
		// 2) 手写 CONNECT（确保请求行是 "CONNECT host:port HTTP/1.1"）
		var b strings.Builder
		b.WriteString("CONNECT ")
		b.WriteString(address)
		b.WriteString(" HTTP/1.1\r\n")
		b.WriteString("Host: ")
		b.WriteString(address)
		b.WriteString("\r\n")

		// Basic auth（如果 proxyURL 里带 userinfo）
		if uri.User != nil {
			u := uri.User.Username()
			p, _ := uri.User.Password()
			auth := base64.StdEncoding.EncodeToString([]byte(u + ":" + p))
			b.WriteString("Proxy-Authorization: Basic ")
			b.WriteString(auth)
			b.WriteString("\r\n")
		}

		b.WriteString("\r\n")

		if _, err = io.WriteString(conn, b.String()); err != nil {
			conn.Close()
			return nil, fmt.Errorf("将CONNECT写入代理失败: %w", err)
		}
		// 3) 读响应（注意：返回带 reader 的 conn，避免缓冲数据丢失）
		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("读取代理响应失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			conn.Close()
			return nil, fmt.Errorf("代理连接失败: %s", resp.Status)
		}
		return conn, nil
	}, nil
}

func NewRemote(raw string) (*Remote, error) {
	var (
		remote = &Remote{}
		uri    *url.URL
		err    error
	)
	if uri, err = url.Parse(raw); err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "socks5", "socks5h":
		if remote.dial, err = NewSocks5(uri); err != nil {
			return nil, err
		}
		return remote, nil
	default:
		if remote.dial, err = NewHttp(uri); err != nil {
			return nil, err
		}
		return remote, nil
	}
}
