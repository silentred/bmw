package bmw

// HandlerRoute is likely deprecated
type HandlerRoute struct {
	Host    string // for api
	Path    string // for api
	Topic   string // for queue (pusher, waiter)
	Handler Handler
}

// type BeanstalkConfig struct {
// 	Host string
// }

type GoChannelConfig struct {
}

func NewSingleBMW(qConfig interface{}, route *HandlerRoute) *MultiBMW {
	var routes []SingleRoute
	handlerRoutes := new(MultiHandlerRoutes)
	handlerRoutes.Host = route.Host

	single := SingleRoute{
		Path:    route.Path,
		Topic:   route.Topic,
		Handler: route.Handler,
	}
	routes = append(routes, single)
	handlerRoutes.Routes = routes

	return NewMultiBMW(qConfig, handlerRoutes)
}

// func (b *SingleBMW) initChanAsQueue(route *HandlerRoute) {
// 	pusherWaiter := queue.NewGoChannelPusherWaiter(1000)

// 	// availablePort := lib.GetPort()
// 	// host := fmt.Sprintf(":%d", availablePort)
// 	apiConfig := &APIConfig{
// 		Host: route.Host,
// 		Path: route.Path,
// 	}

// 	dispatcherConfig := &DispatcherConfig{
// 		MaxWorkers: 4,
// 		WorkerRate: 100,
// 	}

// 	api := NewDefaultAPI(pusherWaiter, apiConfig)
// 	dispatcher := NewDispatcher(route.Handler, pusherWaiter, dispatcherConfig)

// 	b.api = api
// 	b.dispatcher = dispatcher
// }

// func (b *SingleBMW) initBeanstalkAsQueue(config *BeanstalkConfig, route *HandlerRoute) {
// 	beanstalkConfig := &queue.BeanstalkConfig{
// 		Host: config.Host,
// 		Tube: route.Topic,
// 	}
// 	pusherWaiter := queue.NewBeanstalkProducerPusher(beanstalkConfig)

// 	//availablePort := lib.GetPort()
// 	//host := fmt.Sprintf(":%d", availablePort)
// 	apiConfig := &APIConfig{
// 		Host: route.Host,
// 		Path: route.Path,
// 	}

// 	api := NewDefaultAPI(pusherWaiter, apiConfig)
// 	dispatcher := NewDispatcher(route.Handler, pusherWaiter, nil)

// 	b.api = api
// 	b.dispatcher = dispatcher

// }

// func (b *SingleBMW) Start() {
// 	b.dispatcher.Run()
// 	b.api.StartServe()
// }

// // GetAPI return the api
// func (b *SingleBMW) GetAPI() *DefaultAPI {
// 	return b.api
// }
