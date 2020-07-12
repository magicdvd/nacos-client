package nacos

import (
	"encoding/json"
	"fmt"
	"time"
)

type Service struct {
	Dom             string                 `json:"dom"`
	CacheMillis     int64                  `json:"cacheMillis"`
	UseSpecifiedURL bool                   `json:"useSpecifiedUrl"`
	Instances       []*Instance            `json:"hosts"`
	Checksum        string                 `json:"checksum"`
	LastRefTime     int64                  `json:"lastRefTime"`
	Env             string                 `json:"env"`
	Clusters        string                 `json:"clusters"`
	Metadata        map[string]interface{} `json:"metadata"`
	Name            string                 `json:"name"`
	LastUpdateTime  time.Time              `json:"-"`
}

func (c *Service) InstanceDiff(a *Service) bool {
	if len(c.Instances) != len(a.Instances) {
		return true
	}
	cp := make(map[string]*Instance)
	for _, v := range c.Instances {
		cp[fmt.Sprintf("%s:%d", v.Ip, v.Port)] = v
	}
	for _, v := range a.Instances {
		if n, ok := cp[fmt.Sprintf("%s:%d", v.Ip, v.Port)]; !ok {
			return true
		} else {
			if n.Diff(v) {
				return true
			}
		}
	}
	return false
}

type Instance struct {
	Valid       bool              `json:"valid"`
	Marked      bool              `json:"marked"`
	InstanceId  string            `json:"instanceId"`
	Port        uint64            `json:"port"`
	Ip          string            `json:"ip"`
	Weight      float64           `json:"weight"`
	Metadata    map[string]string `json:"metadata"`
	ClusterName string            `json:"clusterName"`
	ServiceName string            `json:"serviceName"`
	Enable      bool              `json:"enabled"`
	Healthy     bool              `json:"healthy"`
	Ephemeral   bool              `json:"ephemeral"`
}

func (c *Instance) Diff(a *Instance) bool {
	if c.Valid != a.Valid {
		return true
	}
	if c.Marked != a.Marked {
		return true
	}
	if c.InstanceId != a.InstanceId {
		return true
	}
	if c.Port != a.Port {
		return true
	}
	if c.Ip != a.Ip {
		return true
	}
	if c.Weight != a.Weight {
		return true
	}
	if c.Metadata != nil && a.Metadata != nil {
		cb, _ := json.Marshal(c.Metadata)
		ab, _ := json.Marshal(a.Metadata)
		if string(cb) != string(ab) {
			return true
		}
	}
	if c.ClusterName != a.ClusterName {
		return true
	}
	if c.ServiceName != a.ServiceName {
		return true
	}
	if c.Enable != a.Enable {
		return true
	}
	if c.Healthy != a.Healthy {
		return true
	}
	if c.Ephemeral != a.Ephemeral {
		return true
	}
	return false
}
