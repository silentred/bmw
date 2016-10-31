package bmw

// Publisher publishes worker ip:port and status to etcd
type Publisher interface {
	Register() error
	Unregister() error
}

// ServiceInfo represents the infomation that Publisher will store at the Registry
type ServiceInfo struct {
	ServiceName string `json:"serviceName"`
	Host        string `json:"host"`
	Privilege   int    `json:"privilege"`
	Key         string `json:"key"`
}
