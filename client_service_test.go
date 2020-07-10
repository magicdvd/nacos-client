package nacos

import (
	"fmt"
	"testing"
	"time"
)

func TestNewServiceClient(t *testing.T) {
	a, err := NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", LogLevel("info"))
	if err != nil {
		t.Error(err)
		return
	}
	err = a.RegisterInstance("127.0.0.1", 8000, "my_test_service")
	if err != nil {
		t.Error(err)
		return
	}
	<-time.After(17 * time.Second)
	t.Log("fin")
}

func TestNewServiceClientGet(t *testing.T) {
	a, err := NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", LogLevel("debug"))
	if err != nil {
		t.Error(err)
		return
	}
	s, err := a.GetService("my_test_service", false, ParamClusters([]string{}))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("clusters", s.Clusters, "lenhosts", len(s.Instances))
	<-time.After(2 * time.Second)
}
