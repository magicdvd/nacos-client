package main

import (
	"fmt"

	"github.com/magicdvd/nacos-client"
)

func main() {
	a, err := nacos.NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", nacos.LogLevel("debug"), nacos.DiscoveryIP("172.21.0.1"))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = a.Watch("my_test_service", func(s *nacos.Service) {
		fmt.Println(s.Instances)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	ch := make(chan bool)
	<-ch
}
