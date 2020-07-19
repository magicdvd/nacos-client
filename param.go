package nacos

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	//service use
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
	keyApp         string = "app"

	//config use
	keyAppName string = "appName"
	keyTenant  string = "tenant"
	keyDataID  string = "dataId"
	keyGroup   string = "group"
	keyContent string = "content"
	keyType    string = "type"
	keyTag     string = "tag"

	keyListenConfigs string = "Listening-Configs"
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
	keys          map[string]bool
	ipAddress     string
	port          uint
	nameSpaceID   string
	weight        float64
	enabled       bool
	healthy       bool
	metadata      map[string]interface{}
	clusterName   string
	serviceName   string
	groupName     string
	ephemeral     bool
	beat          *beatInfo
	clusters      []string
	udpPort       uint
	clientIP      string
	app           string
	appName       string
	tenant        string
	dataID        string
	group         string
	content       string
	tp            string
	tag           string
	listenConfigs string
}

const (
	splitChar1 = string(byte(1))
	splitChar2 = string(byte(2))
)

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

func (c *paramMap) ListenConfigs(content string) {
	if content != "" {
		content = md5string(content)
	}
	c.keys[keyListenConfigs] = true
	delete(c.keys, keyDataID)
	delete(c.keys, keyTenant)
	delete(c.keys, keyGroup)
	if c.tenant != "" {
		c.listenConfigs = fmt.Sprintf("%s%s%s%s%s%s%s%s", c.dataID, splitChar2, c.group, splitChar2, content, splitChar2, c.tenant, splitChar1)
		return
	}
	c.listenConfigs = fmt.Sprintf("%s%s%s%s%s%s", c.dataID, splitChar2, c.group, splitChar2, content, splitChar1)
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
		case keyApp:
			v.Set(k, fmt.Sprint(c.app))
		case keyAppName:
			v.Set(k, fmt.Sprint(c.appName))
		case keyTenant:
			v.Set(k, fmt.Sprint(c.tenant))
		case keyDataID:
			v.Set(k, fmt.Sprint(c.dataID))
		case keyGroup:
			v.Set(k, fmt.Sprint(c.group))
		case keyContent:
			v.Set(k, fmt.Sprint(c.content))
		case keyType:
			v.Set(k, fmt.Sprint(c.tp))
		case keyTag:
			v.Set(k, fmt.Sprint(c.tag))
		case keyListenConfigs:
			v.Set(k, c.listenConfigs)
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

func paramIPAddress(w string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyIPAddress] = true
		m.ipAddress = w
	})
}

func paramServiceName(w string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyServiceName] = true
		m.serviceName = w
	})
}

func paramPort(w uint) Param {
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

func paramBeat(bi *beatInfo) Param {
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

func paramApp(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyApp] = true
		m.app = s
	})
}

func ParamConfigAppName(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyAppName] = true
		m.appName = s
	})
}

func ParamConfigTenant(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyTenant] = true
		m.tenant = s
	})
}

func paramConfigDataID(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyDataID] = true
		m.dataID = s
	})
}

func paramConfigGroup(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyGroup] = true
		m.group = s
	})
}

func paramConfigContent(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyContent] = true
		m.content = s
	})
}

func ParamConfigType(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyType] = true
		m.tp = s
	})
}

func ParamConfigTag(s string) Param {
	return newParam(func(m *paramMap) {
		m.keys[keyTag] = true
		m.tag = s
	})
}
