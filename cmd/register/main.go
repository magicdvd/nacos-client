package main

import (
	"fmt"

	"github.com/magicdvd/nacos-client"
)

func main() {
	a, err := nacos.NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", nacos.LogLevel("info"))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = a.RegisterInstance("172.21.0.1", 8000, "my_test_service", nacos.ParamClusterName("aa"))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = a.RegisterInstance("172.21.0.1", 8000, "my_test_service", nacos.ParamClusterName("bb"))
	if err != nil {
		fmt.Println(err)
		return
	}
	ch := make(chan bool)
	<-ch
}
