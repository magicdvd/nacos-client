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

func TestNewServiceClientPublish(t *testing.T) {
	a, err := NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", LogLevel("debug"))
	if err != nil {
		t.Error(err)
		return
	}
	err = a.PublishConfig("testDataId", "group", "111113")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestNewServiceClientGetConfig(t *testing.T) {
	a, err := NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", LogLevel("debug"))
	if err != nil {
		t.Error(err)
		return
	}
	k, err := a.GetConfig("testDataId", "group")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(k)
}

func TestNewServiceClientRemove(t *testing.T) {
	a, err := NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", LogLevel("debug"))
	if err != nil {
		t.Error(err)
		return
	}
	a.ListenConfig("testDataId", "group", func(s string) {
		t.Log("v", s)
	})
	<-time.After(4 * time.Second)
}
