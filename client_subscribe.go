package nacos

import (
	"encoding/json"
	"errors"
	"fmt"
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
	PushType    string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	LastRefTime int64           `json:"lastRefTime"`
}

func (c *serviceListener) tryListen(port uint) (*net.UDPConn, bool) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
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

func (c *serviceListener) watch(query *paramMap, callback func(*Service), set bool) {
	key := c.buildKey(query.GetGrouppedServiceName(), query.clusters)
	if !set {
		if tmp, ok := c.callbacks.Get(key); ok {
			c.log.Debug("watch add subscribe", key)
			fns := tmp.([]*func(*Service))
			fns = append(fns, &callback)
			c.callbacks.Set(key, fns, cache.NoExpiration)
			return
		}
	}
	c.log.Debug("watch set subscribe", key)
	c.callbacks.Set(key, []*func(*Service){&callback}, cache.NoExpiration)
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

func (c *serviceListener) listen() error {
	var conn *net.UDPConn
	var ok bool
	var port int
	for i := 0; i < 3; i++ {
		//r := rand.New(rand.NewSource(time.Now().UnixNano()))
		port = 54951
		conn, ok = c.tryListen(uint(port))
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
	s, _, err := parseData(data[:n])
	if err != nil {
		c.log.Error("failed to parse push data", err)
		return
	}
	c.log.Debug("receive push ", s, remoteAddr)
	var pushData pushData
	err1 := json.Unmarshal(s, &pushData)
	if err1 != nil {
		c.log.Error("failed to process push data.err", err)
		return
	}
	ack := make(map[string]string)
	if pushData.PushType == "dom" || pushData.PushType == "service" {
		//hosts, err := getServiceHosts(pushData.Data)
		ack["type"] = "push-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	} else if pushData.PushType == "dump" {
		ack["type"] = "dump-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
		b, err := json.Marshal(getServiceMap(c.services))
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
	c.log.Debug("response push ", string(bs), remoteAddr)
}

func (c *ServiceClient) triggerCallback(b []byte) error {
	// c, err := jsonparser.GetString(b, "clusters")
	// if err != nil {
	// 	return err
	// }
	// sv, err := jsonparser.GetString(b, "")

	// hosts, err := getServiceHosts(b)
	// if err != nil {
	// 	return err
	// }
	return nil
}
