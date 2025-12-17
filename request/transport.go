package request

import (
	"fmt"
	"github.com/yydsqu/tools/balancer"
	"github.com/yydsqu/tools/dialer"
	"net/http"
)

type WarpTransport func(parent http.RoundTripper) http.RoundTripper

type RoundRobinProxy struct {
	transports []http.RoundTripper
	round      *balancer.RoundRobin[http.RoundTripper]
}

func (p *RoundRobinProxy) RoundTrip(request *http.Request) (*http.Response, error) {
	return p.round.Next().RoundTrip(request)
}

func RoundRobinTransport(transports ...http.RoundTripper) http.RoundTripper {
	robin, _ := balancer.NewRoundRobin[http.RoundTripper](transports...)
	return &RoundRobinProxy{
		round: robin,
	}
}

func LoadLocalDialerTransport(root *http.Transport, warps ...WarpTransport) (http.RoundTripper, error) {
	locals, err := dialer.LoadLocalDialer()
	if err != nil {
		return nil, fmt.Errorf("加载本地IP信息错误%s", err)
	}
	var transports []http.RoundTripper

	for _, local := range locals {
		transport := root.Clone()
		transport.Dial = local.Dial
		transport.DialContext = local.DialContext
		var tripper http.RoundTripper = transport
		for _, warp := range warps {
			tripper = warp(tripper)
		}
		transports = append(transports, tripper)
	}

	if len(transports) == 0 {
		return nil, fmt.Errorf("没有可用IP")
	}

	return RoundRobinTransport(transports...), nil
}
