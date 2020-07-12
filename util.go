package nacos

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"

	"github.com/buger/jsonparser"
	"github.com/patrickmn/go-cache"
)

const (
	gzBytes1 byte = 0x1f
	gzBytes2 byte = 0x8b
)

func tryGzipDecompress(data []byte) ([]byte, bool, error) {
	if len(data) < 2 || data[0] != gzBytes1 || data[1] != gzBytes2 {
		return data, false, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}
	return bs, true, nil
}

func parseServiceJSON(b []byte) (*Service, error) {
	svc := new(Service)
	err := jsonparser.ObjectEach(b, func(bk []byte, v []byte, t jsonparser.ValueType, offset int) error {
		var err error
		key := string(bk)
		switch key {
		case "dom":
			svc.Dom = string(v)
		case "cacheMillis":
			svc.CacheMillis, err = jsonparser.GetInt(v)
			if err != nil {
				return err
			}
		case "useSpecifiedURL":
			svc.UseSpecifiedURL, err = jsonparser.GetBoolean(v)
			if err != nil {
				return err
			}
		case "hosts":
			svc.Instances = make([]*Instance, 0)
			err = json.Unmarshal(v, &svc.Instances)
			if err != nil {
				return err
			}
		case "checksum":
			svc.Checksum = string(v)
		case "lastRefTime":
			svc.LastRefTime, err = jsonparser.GetInt(v)
			if err != nil {
				return err
			}
		case "env":
			svc.Env = string(v)
		case "clusters":
			svc.Clusters = string(v)
		case "metadata":
			svc.Metadata = make(map[string]interface{})
			err = json.Unmarshal(v, &svc.Metadata)
			if err != nil {
				return err
			}
		case "name":
			svc.Name = string(v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func dumpCache(c *cache.Cache) map[string]interface{} {
	m := make(map[string]interface{})
	items := c.Items()
	for key, it := range items {
		m[key] = it.Object
	}
	return m
}

// func getLocalIP() (string, error) {
// 	addrs, err := net.InterfaceAddrs()
// 	if err != nil {
// 		return "", err
// 	}
// 	for _, address := range addrs {
// 		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
// 			if ipnet.IP.To4() != nil {
// 				return ipnet.IP.String(), nil
// 			}
// 		}
// 	}
// 	return "", errors.New("no local IP")
// }

func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		return "", nil
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if localAddr.IP.To4() != nil {
		return localAddr.IP.String(), nil
	}
	return "", errors.New("no local IP")
}
