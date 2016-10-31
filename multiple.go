package bmw

import "bmw/queue"

type MultiBMW struct {
	api         *DefaultAPI
	dispatchers []*Dispatcher
	routes      *MultiHandlerRoutes
}

type MultiHandlerRoutes struct {
	Host   string
	Routes []SingleRoute
}

type SingleRoute struct {
	Path    string // for api
	Topic   string // for queue (pusher, waiter)
	Handler Handler
}

func NewMultiBMW(qConfig interface{}, routes *MultiHandlerRoutes) *MultiBMW {
	bmw := new(MultiBMW)
	bmw.routes = routes

	if _, ok := qConfig.(GoChannelConfig); ok {
		bmw.initChanAsQueue()
	}

	return bmw
}

func (bmw *MultiBMW) initChanAsQueue() {
	var pusherWaiters []PusherWaiter
	// make RouteConfig
	var routes []*RouteConfig
	for _, item := range bmw.routes.Routes {
		pusherWaiter := queue.NewGoChannelPusherWaiter(1000)
		pusherWaiters = append(pusherWaiters, pusherWaiter)
		dispather := NewDispatcher(item.Handler, pusherWaiter, nil)
		bmw.dispatchers = append(bmw.dispatchers, dispather)

		config := &RouteConfig{path: item.Path, pusher: pusherWaiter}
		routes = append(routes, config)
	}

	api := NewDefaultAPI(bmw.routes.Host, routes)
	bmw.api = api
}

func (b *MultiBMW) Start() {
	for _, item := range b.dispatchers {
		item.Run()
	}
	b.api.StartServe()
}

// GetAPI return the api
func (b *MultiBMW) GetAPI() *DefaultAPI {
	return b.api
}
