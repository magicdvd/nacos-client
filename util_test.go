package nacos

import (
	"net/url"
	"testing"
)

func Test_grouppedServiceName(t *testing.T) {
	a, _ := url.Parse("http://www.domain.com/contextpath")
	t.Log(a.Scheme)
	t.Log(a.User.Username())
	t.Log(a.Hostname())
	t.Log(a.User.Password())
	t.Log(a.Path)
}

func Test_getLocalIP(t *testing.T) {
	got, err := getOutboundIP()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(got)
}
