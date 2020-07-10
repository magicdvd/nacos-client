package nacos

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/magicdvd/nacos-client/constant"
	"github.com/patrickmn/go-cache"
)

// const (
// 	invalidOptionFormat = "invalid option %s"
// )

type ServiceClient struct {
	opts       *clientOptions
	client     *httpClient
	log        LogInterface
	beatMap    *cache.Cache
	lock       sync.Mutex
	nsServices map[string]*serviceListener
}

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
		maxCacheTime:      constant.DefaultMaxCacheTime,
		log:               newDefaultLogger("info"),
		defautNameSpaceID: constant.DefaultNameSpaceID,
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
	if cltOpts.discoveryIP == "" {
		cltOpts.discoveryIP, err = getLocalIP()
		if err != nil {
			return nil, err
		}
	}
	clt := &ServiceClient{
		beatMap:    cache.New(5*time.Minute, 10*time.Minute),
		nsServices: make(map[string]*serviceListener),
		opts:       cltOpts,
		log:        cltOpts.log,
		client:     cltOpts.httpClient,
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
	pm := newParamMap()
	pm.Set(
		ParamIPAddress(ip),
		ParamPort(port),
		ParamServiceName(serviceName),
		ParamEnabled(true),
		ParamWeight(1.0),
		ParamHealthy(true),
		ParamMetadata(map[string]interface{}{}),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(c.opts.defautNameSpaceID),
		ParamClusterName(constant.DefaultClusterName),
		ParamEphemeral(true),
	)
	pm.Set(params...)
	exist, err := c.registerInstance(pm)
	if err != nil {
		return err
	}
	//如果不存在并且临时发送心跳
	if !exist && pm.ephemeral {
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

func (c *ServiceClient) GetService(serviceName string, lazy bool, params ...Param) (*Service, error) {
	query := newParamMap()
	query.Set(
		ParamHealthy(true),
		ParamServiceName(serviceName),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(c.opts.defautNameSpaceID),
	)
	query.Set(params...)
	if lazy {
		svc := c.getCacheService(query.nameSpaceID, query.GetGrouppedServiceName(), query.clusters)
		if svc != nil && time.Since(svc.LastUpdateTime) <= c.opts.maxCacheTime {
			return svc, nil
		}
	}
	return c.getServiceInstances(query)
}

func (c *ServiceClient) setCacheService(nameSpaceID string, grouppedServiceName string, clusters []string, service *Service) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if svc, ok := c.nsServices[nameSpaceID]; ok {
		key := svc.buildKey(grouppedServiceName, clusters)
		svc.services.Set(key, service, cache.NoExpiration)
	} else {
		svc = newServiceListenr(nameSpaceID, c.log)
		key := svc.buildKey(grouppedServiceName, clusters)
		svc.services.Set(key, service, cache.NoExpiration)
		c.nsServices[nameSpaceID] = svc
	}
}

func (c *ServiceClient) getCacheService(namespaceID string, grouppedServiceName string, clusters []string) *Service {
	c.lock.Lock()
	defer c.lock.Unlock()
	if svc, ok := c.nsServices[namespaceID]; ok {
		key := svc.buildKey(grouppedServiceName, clusters)
		if v, ok := svc.services.Get(key); ok {
			sv := v.(*Service)
			return sv
		}
	}
	return nil
}

func (c *ServiceClient) Watch(serviceName string, callback func(*Service), params ...Param) error {
	var svc *serviceListener
	var ok bool
	query := newParamMap()
	query.Set(
		ParamHealthy(true),
		ParamServiceName(serviceName),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(c.opts.defautNameSpaceID),
	)
	query.Set(params...)
	c.lock.Lock()
	if svc, ok = c.nsServices[query.nameSpaceID]; !ok {
		svc = newServiceListenr(query.nameSpaceID, c.log)
	}
	svc.watch(query, callback, true)
	if svc.port == 0 {
		err := svc.listen()
		if err != nil {
			return err
		}
		query.Set(paramUDPPort(svc.port), paramClientIP(c.opts.discoveryIP))
	}
	c.nsServices[query.nameSpaceID] = svc
	c.lock.Unlock()
	_, err := c.getServiceInstances(query)
	if err != nil {
		return err
	}
	return nil
}

func (c *ServiceClient) getServiceInstances(query *paramMap) (*Service, error) {
	b, err := c.client.api(http.MethodGet, constant.APIInstanceList, query, nil)
	if err != nil {
		c.log.Error("GetServiceInstances", "api", err)
		return nil, err
	}
	service, err := getServiceHosts(b)
	if err != nil {
		c.log.Error("GetServiceInstances", "getServiceHosts", err)
		return nil, err
	}
	service.LastUpdateTime = time.Now()
	c.setCacheService(query.nameSpaceID, query.GetGrouppedServiceName(), query.clusters, service)
	return service, nil
}

func (c *ServiceClient) registerInstance(query *paramMap) (bool, error) {
	//serviceName(group@@name):clusterName:ip:port
	k := fmt.Sprintf("%s:%s:%s:%d", query.GetGrouppedServiceName(), query.clusterName, query.ipAddress, query.port)
	if _, exist := c.beatMap.Get(k); exist {
		c.log.Warn("registerInstance", "register duplicate service", query.GetGrouppedServiceName(), query.ipAddress, query.port)
		return true, nil
	}
	c.beatMap.Set(k, true, cache.NoExpiration)
	_, err := c.client.api(http.MethodPost, constant.APIInstance, query, nil)
	if err != nil {
		c.log.Error("registerInstance", "api", err)
		return false, err
	}
	return false, nil
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
		pm := newParamMap()
		pm.Set(ParamIPAddress(beat.IP), ParamPort(beat.Port), ParamServiceName(serviceName), ParamWeight(beat.Weight), ParamMetadata(beat.Metadata), ParamClusterName(beat.Cluster), ParamEphemeral(true), ParamGroupName(groupName))
		_, err := c.registerInstance(pm)
		if err != nil {
			c.log.Error("sendBeat", "re-register", err)
			return err
		}
	}
	return nil
}
