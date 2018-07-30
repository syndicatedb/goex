package proxy

import (
	"net/http"

	"github.com/syndicatedb/goproxy/proxy"
)

type NoProxyProvider struct {
}

func NewNoProxy() proxy.Provider {
	return NoProxyProvider{}
}

func (p NoProxyProvider) NewClient(key string) proxy.Client {
	return &http.Client{}
}
