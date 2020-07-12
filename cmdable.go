package nacos

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
}
