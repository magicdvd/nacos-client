package nacos

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	keyIPAddress   string = "ip"
	keyPort        string = "port"
	keyNameSpaceID string = "namespaceId"
	keyWeight      string = "weight"
	keyEnabled     string = "enabled"
	keyHealthy     string = "healthy"
	keyMetadata    string = "metadata"
	keyClusterName string = "clusterName"
	keyServiceName string = "serviceName"
	keyGroupName   string = "groupName"
	keyEphemeral   string = "ephemeral"
	keyBeat        string = "beat"
	keyClusters    string = "clusters"
	keyUDPPort     string = "udpPort"
	keyClientIP    string = "clientIP"
)

type beatInfo struct {
	IP               string                 `json:"ip"`
	Port             uint                   `json:"port"`
	Weight           float64                `json:"weight"`
	ServiceName      string                 `json:"serviceName"`
	Cluster          string                 `json:"cluster"`
	Metadata         map[string]interface{} `json:"metadata"`
	Scheduled        bool                   `json:"scheduled"`
	LightBeatEnabled bool                   `json:"-"`
	Interval         time.Duration          `json:"-"`
}

func (c *beatInfo) SplitServiceName() (string, string) {
	t := strings.Split(c.ServiceName, "@@")
	return t[0], t[1]
}

type paramMap struct {
	keys        map[string]bool
	ipAddress   string
	port        uint
	nameSpaceID string
	weight      float64
	enabled     bool
	healthy     bool
	metadata    map[string]interface{}
	clusterName string
	serviceName string
	groupName   string
	ephemeral   bool
	beat        *beatInfo
	clusters    []string
	udpPort     uint
	clientIP    string
}

func newParamMap() *paramMap {
	return &paramMap{
		keys: make(map[string]bool),
	}
}

func (c *paramMap) Set(params ...Param) {
	for _, v := range params {
		v.apply(c)
	}
}

func (c *paramMap) GetGrouppedServiceName() string {
	return c.groupName + "@@" + c.serviceName
}

func (c *paramMap) Parse() url.Values {
	v := url.Values{}
	for k := range c.keys {
		switch k {
		case keyIPAddress:
			v.Set(k, c.ipAddress)
		case keyPort:
			v.Set(k, fmt.Sprint(c.port))
		case keyNameSpaceID:
			v.Set(k, c.nameSpaceID)
		case keyWeight:
			v.Set(k, fmt.Sprint(c.weight))
		case keyEnabled:
			if c.enabled {
				v.Set(k, "true")
			} else {
				v.Set(k, "false")
			}
		case keyHealthy:
			if c.healthy {
				v.Set(k, "true")
			} else {
				v.Set(k, "false")
			}
		case keyMetadata:
			b, _ := json.Marshal(c.metadata)
			v.Set(k, string(b))
		case keyClusterName:
			v.Set(k, c.clusterName)
		case keyServiceName:
			v.Set(k, c.serviceName)
		case keyGroupName:
			v.Set(k, c.groupName)
		case keyEphemeral:
			if c.ephemeral {
				v.Set(k, "true")
			} else {
				v.Set(k, "false")
			}
		case keyBeat:
			if c.beat != nil {
				b, _ := json.Marshal(c.beat)
				v.Set(k, string(b))
			}
		case keyClusters:
			if len(c.clusters) > 0 {
				v.Set(k, strings.Join(c.clusters, ","))
			}
		case keyUDPPort:
			v.Set(k, fmt.Sprint(c.udpPort))
		case keyClientIP:
			v.Set(k, fmt.Sprint(c.clientIP))
		}
	}
	return v
}

type Param interface {
	apply(*paramMap)
}

type param struct {
	f func(*paramMap)
}

func (c *param) apply(m *paramMap) {
	c.f(m)
}

func newParam(f func(*paramMap)) Param {
	return &param{
		f: f,
	}
}

func ParamIPAddress(w string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyIPAddress] = true
		m.ipAddress = w
	})
}

func ParamServiceName(w string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyServiceName] = true
		m.serviceName = w
	})
}

func ParamPort(w uint) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyPort] = true
		m.port = w
	})
}

//ParamWeight 权重
func ParamWeight(w float64) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyWeight] = true
		m.weight = w
	})
}

func ParamNameSpaceID(n string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyNameSpaceID] = true
		m.nameSpaceID = n
	})
}

func ParamEnabled(b bool) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyEnabled] = true
		m.enabled = b
	})
}

func ParamHealthy(b bool) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyHealthy] = true
		m.healthy = b
	})
}

func ParamMetadata(mm map[string]interface{}) Param {
	return newParam(func(m *paramMap) {
		if len(mm) > 0 {
			m.keys[keyMetadata] = true
			m.metadata = mm
		}
	})
}

func ParamClusterName(n string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyClusterName] = true
		m.clusterName = n
	})
}

func ParamGroupName(n string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyGroupName] = true
		m.groupName = n
	})
}

func ParamClusters(cs []string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyClusters] = true
		m.clusters = cs
	})
}

func ParamEphemeral(b bool) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyEphemeral] = true
		m.ephemeral = b
	})
}

func ParamBeat(bi *beatInfo) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyBeat] = true
		m.beat = bi
	})
}

func paramUDPPort(port uint) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyUDPPort] = true
		m.udpPort = port
	})
}

func paramClientIP(ip string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyClientIP] = true
		m.clientIP = ip
	})
}
