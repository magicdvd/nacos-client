package main

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	err = a.Subscribe("my_test_service", func(s *nacos.Service) {
		for _, v := range s.Instances {
			fmt.Println("v1", v.Ip, v.Port, v.ServiceName, v.ClusterName, v.Metadata, v.Healthy)
		}
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	<-time.After(time.Minute)
	a.Unsubscribe("my_test_service")
	<-time.After(time.Minute)
	err = a.Subscribe("my_test_service", func(s *nacos.Service) {
		for _, v := range s.Instances {
			fmt.Println("v2", v.Ip, v.Port, v.ServiceName, v.ClusterName, v.Metadata, v.Healthy)
		}
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	ch := make(chan bool)
	<-ch
}
