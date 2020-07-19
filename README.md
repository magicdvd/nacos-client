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
//如果ip不指定,则默认使用 discoveryIP (指定IP/本地可以连接网络的有效IP)
err = a.RegisterInstance("", 8000, "my_test_service", nacos.ParamClusterName("aa"))
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

NewServiceClient(addr string, options ...ClientOption) (ServiceCmdable, error)

```golang
type ServiceCmdable interface {
    //RegisterInstance 注册实例
    RegisterInstance(ip string, port uint, serviceName string, params ...Param) error
    //DeregisterInstance 销毁实例
    DeregisterInstance(ip string, port uint, serviceName string, params ...Param) error
    //GetService 获取服务
    GetService(serviceName string, lazy bool, params ...Param) (*Service, error)
    //Subscribe 订阅
    Subscribe(serviceName string, callback func(*Service), params ...Param) error
    //Unsubscribe 取消订阅
    Unsubscribe(serviceName string, params ...Param)
    //PublishConfig 发布配置
    PublishConfig(dataID string, group string, content string, params ...Param) error
    //GetConfig 获取配置
    GetConfig(dataID string, group string, params ...Param) (string, error)
    //RemoveConfig 获取配置
    RemoveConfig(dataID string, group string, params ...Param) error
    //ListenConfig 监听配置
    ListenConfig(dataID string, group string, callback func(string), params ...Param) <-chan error
}
```

### 客户端选项

- HTTPTimeout 请求超时时间   [15s]
- HTTPTransport http client的transport, 可以支持https, http2等可以在这里自定义,默认使用http1.1的
- LogLevel 日志等级 (debug, info, warn, error) [info]
- Log 设置自定义logger 满足LogInterface即可 [defaultLogger]
- Auth 设置验证user/passwod ["",""]
- MaxCacheTime 服务信息最大缓存时间(影响GetService) [45s]
- DefaultNameSpaceID 设置默认命名空间 [public]
- DiscoveryIP 订阅服务需要服务端推送的IP,多网卡时候可以选定网卡 [本地一个有效IP的地址]
- EnableHTTPRequestLog 是否打开底层http request的日志,配合LogLevel(debug)才生效 [false]
- AppName 订阅时候注册的APPName [app-{DiscoveryIP}]
- DefaultTenant 默认租户信息config使用 [""]

### 功能参数

client.XXXXX(... params ...Param)

Param部分key

|      可选参数      | Service | Config |
| :----------------: | :-----: | :----: |
|    ParamWeight     |    x    |        |
|  ParamNameSpaceID  |    x    |        |
|    ParamEnabled    |    x    |        |
|    ParamHealthy    |    x    |        |
|   ParamMetadata    |    x    |        |
|  ParamClusterName  |    x    |        |
|   ParamGroupName   |    x    |        |
|   ParamClusters    |    x    |        |
|   ParamEphemeral   |    x    |        |
| ParamConfigAppName |         |   x    |
| ParamConfigTenant  |         |   x    |
|  ParamConfigType   |         |   x    |
|   ParamConfigTag   |         |   x    |

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
