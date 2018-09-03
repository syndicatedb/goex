package proxy

import (
	"net"
	"net/http"
	"time"

	"github.com/syndicatedb/goproxy/proxy"
)

type NoProxyProvider struct {
}

func NewNoProxy() proxy.Provider {
	return NoProxyProvider{}
}

var timeout = time.Duration(30 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func (p NoProxyProvider) NewClient(key string) proxy.Client {
	tr := &http.Transport{
		Dial:              dialTimeout,
		DisableKeepAlives: true,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   time.Duration(30 * time.Second),
	}
}

func (p NoProxyProvider) IP() string {
	return ""
}
