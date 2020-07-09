package nacos

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/buger/jsonparser"
	"github.com/magicdvd/nacos-client/constant"
	"github.com/patrickmn/go-cache"
)

// const (
// 	invalidOptionFormat = "invalid option %s"
// )

type ServiceClient struct {
	opts           *clientOptions
	client         *httpClient
	log            LogInterface
	beatMap        *cache.Cache
	serviceCache   *cache.Cache
	subCallbackMap *cache.Cache
}

type serviceInstance struct {
	LastUpdateTime time.Time
	Instances      []*Instance
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

func NewServiceClient(addr string, options ...ClientOption) (*ServiceClient, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	logger := newDefaultLogger("info")
	cltOpts := &clientOptions{
		maxCacheTime: constant.DefaultMaxCacheTime,
		log:          newDefaultLogger("info"),
		httpClient: &httpClient{
			addr:        u.Scheme + "://" + u.Host,
			contextPath: u.Path,
			Client: &http.Client{
				Timeout: constant.DefaultTimeout,
			},
			log: logger,
		},
	}
	if u.User.Username() != "" {
		if pwd, ok := u.User.Password(); ok {
			cltOpts.httpClient.username = u.User.Username()
			cltOpts.httpClient.password = pwd
		}
	}
	l := len(options)
	for i := 0; i < l; i++ {
		op := options[i]
		op.apply(cltOpts)
	}
	clt := &ServiceClient{
		beatMap:      cache.New(5*time.Minute, 10*time.Minute),
		serviceCache: cache.New(cltOpts.maxCacheTime, 2*cltOpts.maxCacheTime),
		opts:         cltOpts,
		log:          cltOpts.log,
		client:       cltOpts.httpClient,
	}
	//配置参数用户名密码配置
	if clt.client.username != "" {
		err = clt.client.login()
		if err != nil {
			return nil, err
		}
		go clt.client.refreshLogin()
		//server url 自带 http://user:passwd@host/contextPath
	}
	return clt, nil
}

func (c *ServiceClient) RegisterInstance(ip string, port uint, serviceName string, params ...Param) error {
	pm, err := c.registerInstance(ip, port, serviceName, params...)
	if err != nil {
		return err
	}
	if pm != nil && pm.ephemeral {
		beat := &beatInfo{
			IP:          pm.ipAddress,
			Port:        pm.port,
			Weight:      pm.weight,
			ServiceName: pm.GetGrouppedServiceName(),
			Cluster:     pm.clusterName,
			Metadata:    pm.metadata,
		}
		nameSpaceID := pm.nameSpaceID
		err = c.sendBeat(nameSpaceID, beat)
		if err != nil {
			return err
		}
		go c.autoSendBeat(nameSpaceID, beat)
	}
	return nil
}

func (c *ServiceClient) GetServiceInstances(serviceName string, lazy bool, params ...Param) ([]*Instance, error) {
	query := newParamMap()
	query.Set(
		ParamHealthy(true),
		ParamServiceName(serviceName),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(constant.DefaultNameSpaceID),
	)
	query.Set(params...)
	key := buildServiceKey(query.GetGrouppedServiceName(), query.nameSpaceID, query.clusters)
	if lazy {
		if v, ok := c.serviceCache.Get(key); ok {
			sv := v.(*serviceInstance)
			if time.Since(sv.LastUpdateTime) <= c.opts.maxCacheTime {
				return sv.Instances, nil
			}
		}
	}
	b, err := c.client.api(http.MethodGet, constant.APIInstanceList, query, nil)
	if err != nil {
		c.log.Error("GetServiceInstances", "api", err)
		return nil, err
	}
	ins, err := getServiceHosts(b)
	if err != nil {
		c.log.Error("GetServiceInstances", "getServiceHosts", err)
		return nil, err
	}
	si := &serviceInstance{
		LastUpdateTime: time.Now(),
		Instances:      ins,
	}
	c.serviceCache.Set(key, si, cache.NoExpiration)
	return ins, nil
}

func (c *ServiceClient) registerInstance(ip string, port uint, serviceName string, params ...Param) (*paramMap, error) {
	query := newParamMap()
	query.Set(
		ParamIPAddress(ip),
		ParamPort(port),
		ParamServiceName(serviceName),
		ParamEnabled(true),
		ParamWeight(1.0),
		ParamHealthy(true),
		ParamMetadata(map[string]interface{}{}),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(constant.DefaultNameSpaceID),
		ParamClusterName(constant.DefaultClusterName),
		ParamEphemeral(true),
	)
	query.Set(params...)
	//serviceName(group@@name):clusterName:ip:port
	k := fmt.Sprintf("%s:%s:%s:%d", query.GetGrouppedServiceName(), query.clusterName, query.ipAddress, query.port)
	if _, exist := c.beatMap.Get(k); exist {
		c.log.Warn("registerInstance", "register duplicate service", query.GetGrouppedServiceName(), query.ipAddress, query.port)
		return nil, nil
	}
	c.beatMap.Set(k, true, cache.NoExpiration)
	_, err := c.client.api(http.MethodPost, constant.APIInstance, query, nil)
	if err != nil {
		c.log.Error("registerInstance", "api", err)
		return nil, err
	}
	return query, nil
}

func (c *ServiceClient) autoSendBeat(nameSpaceID string, beat *beatInfo) {
	for {
		if beat.Interval > 0 {
			<-time.After(beat.Interval)
		}
		_ = c.sendBeat(nameSpaceID, beat)
	}
}

func (c *ServiceClient) sendBeat(nameSpaceID string, beat *beatInfo) error {
	query := newParamMap()
	query.Set(ParamNameSpaceID(nameSpaceID), ParamServiceName(beat.ServiceName), ParamClusterName(beat.Cluster), ParamIPAddress(beat.IP), ParamPort(beat.Port))
	var body *paramMap
	if !beat.LightBeatEnabled {
		body = newParamMap()
		body.Set(ParamBeat(beat))
	}
	b, err := c.client.api(http.MethodPut, constant.APIInstanceBeat, query, body)
	if err != nil {
		c.log.Error("sendBeat", "api", err)
		return err
	}
	interval, err := jsonparser.GetInt(b, constant.ClientBeatInterval)
	if err != nil {
		c.log.Error("sendBeat", "getInterval", err)
		return err
	}
	beat.Interval = time.Duration(interval) * time.Millisecond
	beat.LightBeatEnabled, _ = jsonparser.GetBoolean(b, constant.LightBeatEnabled)
	var code int64
	code, _ = jsonparser.GetInt(b, constant.Code)
	if code == 20404 {
		groupName, serviceName := beat.SplitServiceName()
		_, err := c.registerInstance(beat.IP, beat.Port, serviceName, ParamWeight(beat.Weight), ParamMetadata(beat.Metadata), ParamClusterName(beat.Cluster), ParamEphemeral(true), ParamGroupName(groupName))
		if err != nil {
			c.log.Error("sendBeat", "re-register", err)
			return err
		}
	}
	return nil
}
