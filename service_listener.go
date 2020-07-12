package nacos

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const defaultUDPReadSize int = 4096

type serviceListener struct {
	port        uint
	nameSpaceID string
	services    *cache.Cache
	callbacks   *cache.Cache
	log         LogInterface
}

type pushData struct {
	PushType    string `json:"type"`
	Data        string `json:"data"`
	LastRefTime int64  `json:"lastRefTime"`
}

func (c *serviceListener) tryListen(ip string, port uint) (*net.UDPConn, bool) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		c.log.Error("udp retry", err)
		return nil, false
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		c.log.Error("udp retry", err, port)
		return nil, false
	}
	return conn, true
}

func (c *serviceListener) buildKey(serviceName string, clusters []string) string {
	var cluster string
	if len(clusters) > 0 {
		sort.Strings(clusters)
		cluster = strings.Join(clusters, ",")
		return fmt.Sprintf("%s@@%s", serviceName, cluster)
	}
	return serviceName
}

func (c *serviceListener) subscribe(query *paramMap, callback func(*Service)) {
	key := c.buildKey(query.GetGrouppedServiceName(), query.clusters)
	if tmp, ok := c.callbacks.Get(key); ok {
		c.log.Debug("serviceListener add subscribe", key)
		fns := tmp.([]*func(*Service))
		fns = append(fns, &callback)
		c.callbacks.Set(key, fns, cache.NoExpiration)
		return
	}
	c.callbacks.Set(key, []*func(*Service){&callback}, cache.NoExpiration)
}

func (c *serviceListener) unsubscribe(query *paramMap) {
	key := c.buildKey(query.GetGrouppedServiceName(), query.clusters)
	c.callbacks.Delete(key)
}

func (c *serviceListener) isSubscribed(query *paramMap) bool {
	key := c.buildKey(query.GetGrouppedServiceName(), query.clusters)
	_, ok := c.callbacks.Get(key)
	return ok
}

func newServiceListenr(nameSpaceID string, log LogInterface) *serviceListener {
	pr := &serviceListener{
		nameSpaceID: nameSpaceID,
		callbacks:   cache.New(5*time.Minute, 10*time.Minute),
		services:    cache.New(5*time.Minute, 10*time.Minute),
		log:         log,
	}
	return pr
}

func (c *serviceListener) listen(addr string) error {
	var conn *net.UDPConn
	var ok bool
	var port int
	for i := 0; i < 3; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		port = r.Intn(1000) + 54951
		conn, ok = c.tryListen(addr, uint(port))
		if ok {
			c.log.Debug("start udp server listen", port)
			break
		}
		if !ok && i == 2 {
			return errors.New("failed to start udp server after trying 3 times.")
		}
	}
	c.port = uint(port)
	go func() {
		defer conn.Close()
		for {
			c.handleClient(conn)
		}
	}()
	return nil
}

func (c *serviceListener) handleClient(conn *net.UDPConn) {
	data := make([]byte, defaultUDPReadSize)
	n, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		c.log.Error("failed to read UDP msg", err)
		return
	}
	s, _, err := tryGzipDecompress(data[:n])
	if err != nil {
		c.log.Error("failed to parse push data", err)
		return
	}
	c.log.Debug("recieve push", string(s), remoteAddr)
	var pushData pushData
	err1 := json.Unmarshal(s, &pushData)
	if err1 != nil {
		c.log.Error("failed to process push data", err)
		return
	}
	ack := make(map[string]string)
	if pushData.PushType == "dom" || pushData.PushType == "service" {
		service, err := parseServiceJSON([]byte(pushData.Data))
		if err != nil {
			c.log.Error("recieve push data error", c.nameSpaceID, err)
		} else {
			clusters := make([]string, 0)
			if service.Clusters != "" {
				clusters = strings.Split(service.Clusters, ",")
			}
			key := c.buildKey(service.Name, clusters)
			if v, ok := c.services.Get(key); !ok {
				c.triggerCallback(key, service)
			} else {
				sv := v.(*Service)
				if service.InstanceDiff(sv) {
					c.triggerCallback(key, service)
				}
			}
		}
		ack["type"] = "push-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	} else if pushData.PushType == "dump" {
		ack["type"] = "dump-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
		b, err := json.Marshal(dumpCache(c.services))
		if err != nil {
			c.log.Error("dump service map error", c.nameSpaceID, err)
		} else {
			ack["data"] = string(b)
		}
	} else {
		ack["type"] = "unknow-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	}
	bs, _ := json.Marshal(ack)
	_, _ = conn.WriteToUDP(bs, remoteAddr)
	c.log.Debug("push write back", string(bs), remoteAddr)
}

func (c *serviceListener) triggerCallback(key string, s *Service) {
	if tmp, ok := c.callbacks.Get(key); ok {
		fns := tmp.([]*func(*Service))
		for _, v := range fns {
			fn := *v
			go fn(s)
		}
	}
}
