# nacos client

nacos-sdk-go 功能很全, 有些功能没有lazy load,所以为满足目前业务需要,实现namingClient的功能。
目前开发阶段,以下描述的功能都已完成,但未做完整单元测试

## 服务注册/销毁

```golang
a, err := nacos.NewServiceClient("http://nacos:nacos@127.0.0.1:8848/nacos", nacos.LogLevel("debug"), nacos.EnableHTTPRequestLog(false))
if err != nil {
    return
}
err = a.RegisterInstance("172.21.0.1", 8000, "my_test_service", nacos.ParamClusterName("aa"))
if err != nil {
    return
}
err = a.DeregisterInstance("172.21.0.1", 8000, "my_test_service", nacos.ParamClusterName("aa"))
if err != nil {
    return
}
```

## 获取服务

```golang
service, err = a.GetService("my_test_service", true)
if err != nil {
    return
}
```

第2个参数, 是用来决定是否是从再maxCacheTime内取cache,还是直接去服务端获取,建议用true

## 服务订阅

```golang
err = a.Subscribe("my_test_service", func(s *nacos.Service) {
    for _, v := range s.Instances {
        fmt.Println("v1", v.Ip, v.Port, v.ServiceName, v.ClusterName, v.Metadata, v.Healthy)
    }
})

a.Unsubscribe("my_test_service")
```

## 参数说明

NewServiceClient(addr string, options ...ClientOption)

### 客户端选项

- HTTPTimeout 请求超时时间   [15s]
- HTTPTransport http client的transport, 可以支持https, http2等可以在这里自定义,默认使用http1.1的
- LogLevel 日志等级 (debug, info, warn, error) [info]
- Log 设置自定义logger 满足LogInterface即可 [defaultLogger]
- Auth 设置验证user/passwod ["",""]
- MaxCacheTime 服务信息最大缓存时间(影响GetService) [45s]
- DefaultNameSpaceID 设置默认命名空间 [public]
- DiscoveryIP 订阅服务需要服务端推送的IP,多网卡时候可以选定网卡 [本地网卡所有可用地址]
- EnableHTTPRequestLog 是否打开底层http request的日志,配合LogLevel(debug)才生效 [false]
- AppName 订阅时候注册的APPName [app-{DiscoveryIP}]

### 功能参数

client.XXXXX(... params ...Param)

Param部分key

```golang
keyIPAddress   string = "ip"
keyPort        string = "port"
keyNameSpaceID string = "namespaceId"
keyWeight      string = "weight"
keyEnabled     string = "enabled"
keyHealthy     string = "healthy"
keyMetadata    string = "metadata"
keyClusterName string = "clusterName"
keyServiceName string = "serviceName"
keyGroupName   string = "groupName"
keyEphemeral   string = "ephemeral"
keyBeat        string = "beat"
keyClusters    string = "clusters"
keyUDPPort     string = "udpPort"
keyClientIP    string = "clientIP"
keyApp         string = "app"
```

## 其他

nacos-docker目录是用来做测试的

如果要测试subscribe有些麻烦

先编译(以下是linux的,如果是win,mac得另外编译成linux的程序)

```shell
# build subscriber
cd cmd/subscribe
go build -o subscribe main.go
```

```shell
# 启动docker-compose
docker-compose -f nacos-docker/standalone-derby.yaml up -d
#查看subscribe的日志
docker-compose -f nacos-docker/standalone-derby.yaml logs -f subscriber
```
