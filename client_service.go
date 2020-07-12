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
			enableLog: false,
			log:       logger,
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
	if cltOpts.appName == "" {
		cltOpts.appName = "app-" + cltOpts.discoveryIP
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
		ParamNameSpaceID(c.opts.defautNameSpaceID),
		ParamClusterName(constant.DefaultClusterName),
		ParamEphemeral(true),
	)
	query.Set(params...)
	if c.existBeatMap(query.ipAddress, query.port, query.serviceName, query.groupName, query.nameSpaceID, query.clusterName) {
		c.log.Warn("register duplicate service")
		return nil
	}
	c.registerBeatMap(query.ipAddress, query.port, query.serviceName, query.groupName, query.nameSpaceID, query.clusterName)
	exist, err := c.registerInstance(query, true)
	if err != nil {
		return err
	}
	//如果不存在并且临时发送心跳
	if !exist && query.ephemeral {
		beat := &beatInfo{
			IP:          query.ipAddress,
			Port:        query.port,
			Weight:      query.weight,
			ServiceName: query.GetGrouppedServiceName(),
			Cluster:     query.clusterName,
			Metadata:    query.metadata,
		}
		nameSpaceID := query.nameSpaceID
		err = c.sendBeat(nameSpaceID, beat)
		if err != nil {
			return err
		}
		go c.autoSendBeat(nameSpaceID, beat)
	}
	return nil
}

func (c *ServiceClient) DeregisterInstance(ip string, port uint, serviceName string, params ...Param) error {
	query := newParamMap()
	query.Set(
		ParamIPAddress(ip),
		ParamPort(port),
		ParamServiceName(serviceName),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(c.opts.defautNameSpaceID),
		ParamClusterName(constant.DefaultClusterName),
		ParamEphemeral(true),
	)
	query.Set(params...)
	c.deregisterBeatMap(query.ipAddress, query.port, query.nameSpaceID, query.groupName, query.serviceName, query.clusterName)
	c.log.Debug(fmt.Sprintf("deregister instance serviceName:%s, group: %s, cluster: %s, ip: %s, port: %d, namespaceid: %s", query.serviceName, query.groupName, query.clusterName, query.ipAddress, query.port, query.nameSpaceID))
	_, err := c.client.api(http.MethodDelete, constant.APIInstance, query, nil)
	if err != nil {
		c.log.Error("deregisterInstance", "api", err)
		return err
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

func (c *ServiceClient) Subscribe(serviceName string, callback func(*Service), params ...Param) error {
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
	c.log.Debug(fmt.Sprintf("subscribe service:%s, group: %s, clusters: %s, namespaceid: %s", query.serviceName, query.groupName, query.clusters, query.nameSpaceID))
	svc.subscribe(query, callback)
	if svc.port == 0 {
		err := svc.listen(c.opts.discoveryIP)
		if err != nil {
			return err
		}
	}
	query.Set(paramUDPPort(svc.port), paramClientIP(c.opts.discoveryIP), paramApp(c.opts.appName))
	c.nsServices[query.nameSpaceID] = svc
	c.lock.Unlock()
	service, err := c.getServiceInstances(query)
	if err != nil {
		return err
	}
	lastLocalRefreshTime := time.Now()
	var lastRefTime time.Time
	var pastTime time.Duration
	cacheTime := constant.DefaultSubscrubeCacheTime
	go func() {
		for {
			lastRefTime = lastLocalRefreshTime
			if service.LastRefTime > 0 && service.CacheMillis > 0 {
				//使用服务器刷新时间
				cacheTime = time.Duration(service.CacheMillis) * time.Millisecond
				lastRefTime = time.Unix(0, int64(time.Millisecond)*service.LastRefTime)
			}
			pastTime = time.Since(lastRefTime) - cacheTime
			if pastTime >= 0 {
				lastLocalRefreshTime = time.Now()
				service, err = c.getServiceInstances(query)
				if err != nil {
					c.log.Error("watch service, sync push service error", err)
				}
			} else {
				<-time.After(-pastTime)
			}
			//取消订阅，则停止心跳
			if !svc.isSubscribed(query) {
				return
			}
		}
	}()
	return nil
}

func (c *ServiceClient) Unsubscribe(serviceName string, params ...Param) {
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
	defer c.lock.Unlock()
	if svc, ok = c.nsServices[query.nameSpaceID]; !ok {
		return
	}
	c.log.Debug(fmt.Sprintf("unsubscribe service:%s, group: %s, clusters: %s, namespaceid: %s", query.serviceName, query.groupName, query.clusters, query.nameSpaceID))
	svc.unsubscribe(query)
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

func (c *ServiceClient) getServiceInstances(query *paramMap) (*Service, error) {
	b, err := c.client.api(http.MethodGet, constant.APIInstanceList, query, nil)
	if err != nil {
		c.log.Error("GetServiceInstances", "api", err)
		return nil, err
	}
	service, err := parseServiceJSON(b)
	if err != nil {
		c.log.Error("GetServiceInstances", "getServiceHosts", err)
		return nil, err
	}
	service.LastUpdateTime = time.Now()
	c.setCacheService(query.nameSpaceID, query.GetGrouppedServiceName(), query.clusters, service)
	return service, nil
}

func (c *ServiceClient) existBeatMap(ip string, port uint, serviceName string, groupName string, nameSpaceID string, clusterName string) bool {
	k := fmt.Sprintf("%s:%s:%s:%s:%s:%d", serviceName, groupName, nameSpaceID, clusterName, ip, port)
	_, exist := c.beatMap.Get(k)
	return exist
}

func (c *ServiceClient) registerBeatMap(ip string, port uint, serviceName string, groupName string, nameSpaceID string, clusterName string) {
	k := fmt.Sprintf("%s:%s:%s:%s:%s:%d", serviceName, groupName, nameSpaceID, clusterName, ip, port)
	c.beatMap.Set(k, true, cache.NoExpiration)
}

func (c *ServiceClient) deregisterBeatMap(ip string, port uint, nameSpaceID string, groupName string, serviceName string, clusterName string) {
	k := fmt.Sprintf("%s:%s:%s:%s:%s:%d", serviceName, groupName, nameSpaceID, clusterName, ip, port)
	c.beatMap.Delete(k)
}

func (c *ServiceClient) registerInstance(query *paramMap, setBeat bool) (bool, error) {
	c.log.Debug(fmt.Sprintf("register instance serviceName:%s, group: %s, cluster: %s, ip: %s, port: %d, namespaceid: %s", query.serviceName, query.groupName, query.clusterName, query.ipAddress, query.port, query.nameSpaceID))
	_, err := c.client.api(http.MethodPost, constant.APIInstance, query, nil)
	if err != nil {
		c.log.Error("registerInstance", "api", err)
		return false, err
	}
	return false, nil
}

func (c *ServiceClient) autoSendBeat(nameSpaceID string, beat *beatInfo) {
	groupName, serviceName := beat.SplitServiceName()
	for {
		if beat.Interval > 0 {
			<-time.After(beat.Interval)
		}
		//已经注销
		if !c.existBeatMap(beat.IP, beat.Port, serviceName, groupName, nameSpaceID, beat.Cluster) {
			return
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
		//已经注销
		if !c.existBeatMap(query.ipAddress, query.port, query.serviceName, query.groupName, query.nameSpaceID, query.clusterName) {
			return nil
		}
		_, err := c.registerInstance(pm, false)
		if err != nil {
			c.log.Error("sendBeat", "re-register", err)
			return err
		}
	}
	return nil
}
