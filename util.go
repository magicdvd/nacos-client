package nacos

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/buger/jsonparser"
)

const (
	gzBytes1 byte = 0x1f
	gzBytes2 byte = 0x8b
)

func parseData(data []byte) ([]byte, bool, error) {
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

func buildServiceKey(serviceName string, namespaceID string, clusters []string) string {
	var cluster string
	if len(clusters) > 0 {
		sort.Strings(clusters)
		cluster = strings.Join(clusters, ",")
	}
	return fmt.Sprintf("%s:%s:%s", serviceName, namespaceID, cluster)
}

func getServiceHosts(b []byte) ([]*Instance, error) {
	data, _, _, err := jsonparser.Get(b, "hosts")
	if err != nil {
		return nil, err
	}
	var ins []*Instance
	err = json.Unmarshal(data, &ins)
	if err != nil {
		return nil, err
	}
	return ins, nil
}
