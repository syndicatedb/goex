package main

import (
	"fmt"
	"time"

	"github.com/syndicatedb/goex/exchanges"
	"github.com/syndicatedb/goex/schemas"
	"github.com/syndicatedb/goproxy/proxy"
)

const (
	proxyServer = "http://:8081"
)

var (
	chs chan schemas.Result
)

func init() {
	httpProxyAgent := proxy.New(proxyServer)
	ex := exchanges.NewPublic(exchanges.Tidex)
	ex.SetProxyProvider(httpProxyAgent)
	ex.InitProviders()

	chs = ex.GetSymbolProvider().Subscribe(1 * time.Hour)
}

func main() {
	for msg := range chs {
		fmt.Println("msg: ", msg)
	}
}
