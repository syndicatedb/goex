package main

import (
	runner "github.com/niklucky/go-runner"
	"github.com/syndicatedb/goex/exchanges"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	proxyServer = "http://:8081"
)

var (
	worker Worker
)

func init() {
	// Init HTTP Proxy provider
	httpProxyProvider := proxy.New(proxyServer)
	worker = NewWorker(exchanges.Tidex, httpProxyProvider)
}

func main() {
	r := runner.New()
	r.Add(&worker)
	r.Run()
}
