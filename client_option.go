package nacos

import (
	"net/http"
	"time"
)

type clientOptions struct {
	maxCacheTime      time.Duration
	log               LogInterface
	httpClient        *httpClient
	listenInterval    time.Duration
	defautNameSpaceID string
	defaultTenant     string
	discoveryIP       string
	appName           string
}

type ClientOption interface {
	apply(*clientOptions)
}

type funcClientOption struct {
	f func(*clientOptions)
}

func newFuncClientOption(f func(*clientOptions)) *funcClientOption {
	return &funcClientOption{
		f: f,
	}
}

func (fco *funcClientOption) apply(do *clientOptions) {
	fco.f(do)
}

func HTTPTimeout(s time.Duration) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.httpClient.client.Timeout = s
	})
}

func ListenInterval(s time.Duration) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.httpClient.listenClient.Timeout = s + 10*time.Second
		o.listenInterval = s
	})
}

func Log(log LogInterface) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.log = log
		o.httpClient.log = o.log
	})
}

func LogLevel(s string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.log.SetLevel(s)
		o.httpClient.log.SetLevel(s)
	})
}

// Creds returns a ServerOption that sets credentials for server connections.
func HTTPTransport(c http.RoundTripper) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.httpClient.client.Transport = c
		o.httpClient.listenClient.Transport = c
	})
}

func Auth(user, password string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.httpClient.username = user
		o.httpClient.password = password
	})
}

func MaxCacheTime(s time.Duration) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.maxCacheTime = s
	})
}

func DefaultNameSpaceID(s string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		if s != "" {
			o.defautNameSpaceID = s
		}
	})
}

func DiscoveryIP(s string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		if s != "" {
			o.discoveryIP = s
		}
	})
}

func EnableHTTPRequestLog(b bool) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.httpClient.enableLog = b
	})
}

func AppName(s string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.appName = s
	})
}

func DefaultTenant(s string) ClientOption {
	return newFuncClientOption(func(o *clientOptions) {
		o.defaultTenant = s
	})
}
