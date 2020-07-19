package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/magicdvd/nacos-client"
)

const (
	EnvNacosServerAddr    = "NACOS_SERVER"
	EnvLogLevel           = "LOG_LEVEL"
	EnvEnableDebugRequest = "ENABLE_REQUEST_LOG"
)

func main() {
	addr := os.Getenv(EnvNacosServerAddr)
	if addr == "" {
		addr = "http://nacos:nacos@127.0.0.1:8848/nacos"
	}
	ll := os.Getenv(EnvLogLevel)
	if ll == "" {
		ll = "debug"
	}
	debug := os.Getenv(EnvEnableDebugRequest)
	d := false
	if strings.EqualFold(debug, "yes") || strings.EqualFold(debug, "true") {
		d = true
	}
	a, err := nacos.NewServiceClient(addr, nacos.LogLevel(ll), nacos.EnableHTTPRequestLog(d))
	if err != nil {
		fmt.Println(err)
		return
	}
	a.ListenConfig("testDataId", "group", func(s string) {
		fmt.Println("value change", s)
	})
	ch := make(chan bool)
	<-ch
}
