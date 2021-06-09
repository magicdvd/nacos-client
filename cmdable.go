package nacos

type ServiceCmdable interface {
	//RegisterInstance 注册实例
	RegisterInstance(ip string, port uint, serviceName string, params ...Param) error
	//HeartBeatErr 注册实例如果出错时候的回调
	HeartBeatErr() <-chan error
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
