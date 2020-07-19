package constant

import "time"

const (
	ClientVersion = "nacos-go-sdk:v1.0.0"

	DefaultClusterName        = "DEFAULT"
	DefaultGroupName          = "DEFAULT_GROUP"
	DefaultNameSpaceID        = "public"
	DefaultTimeout            = 15 * time.Second
	DefaultListenInterval     = 30 * time.Second
	DefaultMaxCacheTime       = 45 * time.Second
	DefaultSubscrubeCacheTime = 10 * time.Second

	AccessToken        = "accessToken"
	AccessTokenTTL     = "tokenTtl"
	LightBeatEnabled   = "lightBeatEnabled"
	ClientBeatInterval = "clientBeatInterval"
	Code               = "code"

	APILoginPath    = "/v1/auth/users/login"
	APIInstance     = "/v1/ns/instance"
	APIInstanceList = "/v1/ns/instance/list"
	APIInstanceBeat = "/v1/ns/instance/beat"

	APIConfig       = "/v1/cs/configs"
	APIConfigListen = "/v1/cs/configs/listener"
)
