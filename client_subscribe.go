package nacos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/magicdvd/nacos-client/constant"
	"github.com/patrickmn/go-cache"
)

const defaultUDPReadSize int = 4096

type pushData struct {
	PushType    string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	LastRefTime int64           `json:"lastRefTime"`
}

func (c *ServiceClient) tryListen(port uint) (*net.UDPConn, bool) {
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

func (c *ServiceClient) Watch(serviceName string, callback func([]*Instance), params ...Param) {
	c.watch(serviceName, callback, true, params...)
}

func (c *ServiceClient) AddWatch(serviceName string, callback func([]*Instance), params ...Param) {
	c.watch(serviceName, callback, false, params...)
}

func (c *ServiceClient) watch(serviceName string, callback func([]*Instance), set bool, params ...Param) {
	query := newParamMap()
	query.Set(
		ParamHealthy(true),
		ParamServiceName(serviceName),
		ParamGroupName(constant.DefaultGroupName),
		ParamNameSpaceID(constant.DefaultNameSpaceID),
	)
	query.Set(params...)
	key := buildServiceKey(query.GetGrouppedServiceName(), query.nameSpaceID, query.clusters)
	if !set {
		if tmp, ok := c.subCallbackMap.Get(key); ok {
			c.log.Debug("watch add subscribe", key)
			fns := tmp.([]*func([]*Instance))
			fns = append(fns, &callback)
			c.subCallbackMap.Set(key, fns, cache.NoExpiration)
			return
		}
	}
	c.log.Debug("watch set subscribe", key)
	c.subCallbackMap.Set(key, []*func([]*Instance){&callback}, cache.NoExpiration)
}

func (c *ServiceClient) SubscribeStart(ctx context.Context) error {
	var conn *net.UDPConn
	for i := 0; i < 3; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		port := r.Intn(1000) + 54951
		conn1, ok := c.tryListen(uint(port))
		if ok {
			conn = conn1
			c.log.Info(fmt.Sprintf("udp server start, port: %d", port))
			break
		}
		if !ok && i == 2 {
			return errors.New("failed to start udp server after trying 3 times.")
		}
	}
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		c.handleClient(conn)
	}
}

func (c *ServiceClient) handleClient(conn *net.UDPConn) {
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
		log.Printf("[ERROR] failed to process push data.err:%s \n", err1.Error())
		return
	}
	ack := make(map[string]string)

	if pushData.PushType == "dom" || pushData.PushType == "service" {
		//hosts, err := getServiceHosts(pushData.Data)
		ack["type"] = "push-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	} else if pushData.PushType == "dump" {
		// ack["type"] = "dump-ack"
		// ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		// ack["data"] = utils.ToJsonString(us.hostReactor.serviceInfoMap)
		return
	} else {
		ack["type"] = "unknow-ack"
		ack["lastRefTime"] = strconv.FormatInt(pushData.LastRefTime, 10)
		ack["data"] = ""
	}

	bs, _ := json.Marshal(ack)
	_, _ = conn.WriteToUDP(bs, remoteAddr)
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
